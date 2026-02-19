package attachment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	uploadDomain "github.com/mwork/mwork-api/internal/domain/upload"
)

var (
	ErrAttachmentNotFound = errors.New("attachment not found")
	ErrNotOwner           = errors.New("not the owner of this attachment")
)

// Service handles attachment business logic.
type Service struct {
	repo          Repository
	uploadService *uploadDomain.Service
}

// NewService creates an attachment service.
func NewService(repo Repository, uploadService *uploadDomain.Service) *Service {
	return &Service{
		repo:          repo,
		uploadService: uploadService,
	}
}

// Attach links an existing upload to a business entity.
// The caller provides the upload_id (from POST /files), the target, and optional metadata.
func (s *Service) Attach(
	ctx context.Context,
	uploadID uuid.UUID,
	callerID uuid.UUID,
	targetType TargetType,
	targetID uuid.UUID,
	metadata Metadata,
) (*AttachmentWithURL, error) {
	// Verify upload exists and belongs to caller
	upload, err := s.uploadService.GetByID(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("upload not found: %w", err)
	}
	if upload.AuthorID != callerID {
		return nil, ErrNotOwner
	}

	// Get current count for sort_order
	count, err := s.repo.CountByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("count attachments: %w", err)
	}

	a := &Attachment{
		ID:         uuid.New(),
		UploadID:   uploadID,
		TargetID:   targetID,
		TargetType: targetType,
		SortOrder:  count,
		Metadata:   metadata,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create attachment: %w", err)
	}

	return s.enrich(a, upload), nil
}

// ListByTarget returns all attachments for an entity with enriched URL info.
func (s *Service) ListByTarget(ctx context.Context, targetType TargetType, targetID uuid.UUID) ([]*AttachmentWithURL, error) {
	attachments, err := s.repo.ListByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, err
	}

	result := make([]*AttachmentWithURL, 0, len(attachments))
	for _, a := range attachments {
		upload, err := s.uploadService.GetByID(ctx, a.UploadID)
		if err != nil {
			continue // Skip orphaned attachments gracefully
		}
		result = append(result, s.enrich(a, upload))
	}
	return result, nil
}

// GetByID returns a single attachment with URL.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*AttachmentWithURL, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAttachmentNotFound
	}
	upload, err := s.uploadService.GetByID(ctx, a.UploadID)
	if err != nil {
		return nil, err
	}
	return s.enrich(a, upload), nil
}

// Delete removes an attachment. The underlying file (upload) is NOT deleted —
// uploads are immutable warehouse items. Use DELETE /files/{id} to delete the file.
func (s *Service) Delete(ctx context.Context, id uuid.UUID, callerID uuid.UUID) error {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrAttachmentNotFound
	}

	// Verify ownership via upload's author_id
	upload, err := s.uploadService.GetByID(ctx, a.UploadID)
	if err != nil {
		return err
	}
	if upload.AuthorID != callerID {
		return ErrNotOwner
	}

	return s.repo.Delete(ctx, id)
}

// Reorder updates sort_order for a list of attachment IDs.
func (s *Service) Reorder(ctx context.Context, ids []uuid.UUID) error {
	for i, id := range ids {
		if err := s.repo.UpdateSortOrder(ctx, id, i); err != nil {
			return fmt.Errorf("update sort order for %s: %w", id, err)
		}
	}
	return nil
}

// enrich combines an Attachment with its upload's URL and file info.
func (s *Service) enrich(a *Attachment, upload *uploadDomain.Upload) *AttachmentWithURL {
	return &AttachmentWithURL{
		Attachment:   *a,
		URL:          s.uploadService.GetURL(upload),
		OriginalName: upload.OriginalName,
		MimeType:     upload.MimeType,
		SizeBytes:    upload.SizeBytes,
	}
}
