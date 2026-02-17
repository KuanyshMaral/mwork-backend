package upload

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/pkg/storage"
)

// Batch upload methods

// StageBatch uploads multiple files as a batch
func (s *Service) StageBatch(ctx context.Context, userID uuid.UUID, batchID uuid.UUID, files []struct {
	Filename string
	Reader   io.Reader
	Purpose  string
}) ([]*Upload, error) {
	if len(files) == 0 {
		return nil, errors.New("no files provided")
	}

	uploads := make([]*Upload, 0, len(files))
	now := time.Now()

	for _, file := range files {
		// Determine purpose/category
		purpose := file.Purpose
		if purpose == "" {
			purpose = string(CategoryPhoto)
		}

		// Validate file
		buffer, mimeType, err := storage.ValidateAndBuffer(file.Reader, purpose)
		if err != nil {
			return nil, fmt.Errorf("validation failed for %s: %w", file.Filename, err)
		}

		// Generate unique key
		uploadID := uuid.New()
		ext := storage.GetExtensionForMime(mimeType)
		stagingKey := fmt.Sprintf("staging/%s/%s%s", userID.String(), uploadID.String(), ext)
		sizeBytes := int64(buffer.Len())

		// Store in staging
		if err := s.stagingStorage.Put(ctx, stagingKey, buffer, mimeType); err != nil {
			return nil, fmt.Errorf("failed to store file %s: %w", file.Filename, err)
		}

		// Create upload record
		upload := &Upload{
			ID:           uploadID,
			UserID:       userID,
			Category:     Category(purpose),
			Purpose:      purpose,
			Status:       StatusStaged,
			OriginalName: filepath.Base(file.Filename),
			MimeType:     mimeType,
			Size:         sql.NullInt64{Int64: sizeBytes, Valid: true},
			StagingKey:   stagingKey,
			BatchID:      uuid.NullUUID{UUID: batchID, Valid: true},
			CreatedAt:    now,
			ExpiresAt:    now.Add(s.config.StagingTTL),
		}

		uploads = append(uploads, upload)
	}

	// Create all records in batch
	if err := s.repo.CreateBatch(ctx, uploads); err != nil {
		// Cleanup staging files on DB error
		for _, upload := range uploads {
			_ = s.stagingStorage.Delete(ctx, upload.StagingKey)
		}
		return nil, fmt.Errorf("failed to create batch upload records: %w", err)
	}

	return uploads, nil
}

// CommitBatch commits all uploads in a batch
func (s *Service) CommitBatch(ctx context.Context, userID uuid.UUID, batchID uuid.UUID) ([]*Upload, error) {
	uploads, err := s.repo.GetByBatchID(ctx, batchID)
	if err != nil {
		return nil, err
	}

	if len(uploads) == 0 {
		return nil, ErrUploadNotFound
	}

	// Verify ownership (check first upload)
	if uploads[0].UserID != userID {
		return nil, ErrNotUploadOwner
	}

	committed := make([]*Upload, 0, len(uploads))

	for _, upload := range uploads {
		// Skip if already committed
		if upload.IsCommitted() {
			committed = append(committed, upload)
			continue
		}

		// Commit individual upload
		committedUpload, err := s.Commit(ctx, upload.ID, userID)
		if err != nil {
			log.Error().Err(err).Str("upload_id", upload.ID.String()).Msg("failed to commit upload in batch")
			continue
		}

		committed = append(committed, committedUpload)
	}

	return committed, nil
}

// GetBatch retrieves all uploads in a batch
func (s *Service) GetBatch(ctx context.Context, userID uuid.UUID, batchID uuid.UUID) ([]*Upload, error) {
	uploads, err := s.repo.GetByBatchID(ctx, batchID)
	if err != nil {
		return nil, err
	}

	if len(uploads) == 0 {
		return nil, ErrUploadNotFound
	}

	// Verify ownership
	if uploads[0].UserID != userID {
		return nil, ErrNotUploadOwner
	}

	return uploads, nil
}
