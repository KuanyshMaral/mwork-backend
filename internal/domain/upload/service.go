package upload

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/pkg/storage"
)

const (
	StagingTTL = 1 * time.Hour // Files expire after 1 hour if not committed
)

// Service handles upload business logic
type Service struct {
	repo           Repository
	stagingStorage storage.Storage
	cloudStorage   storage.Storage // nil if cloud not configured
	stagingBaseURL string
}

// NewService creates upload service
func NewService(repo Repository, stagingStorage storage.Storage, cloudStorage storage.Storage, stagingBaseURL string) *Service {
	return &Service{
		repo:           repo,
		stagingStorage: stagingStorage,
		cloudStorage:   cloudStorage,
		stagingBaseURL: stagingBaseURL,
	}
}

// Stage uploads a file to staging storage
func (s *Service) Stage(ctx context.Context, userID uuid.UUID, category Category, filename string, reader io.Reader) (*Upload, error) {
	// Validate file
	buffer, mimeType, err := storage.ValidateAndBuffer(reader, string(category))
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Generate unique key
	uploadID := uuid.New()
	ext := storage.GetExtensionForMime(mimeType)
	stagingKey := fmt.Sprintf("staging/%s/%s%s", userID.String(), uploadID.String(), ext)

	// Store in staging
	if err := s.stagingStorage.Put(ctx, stagingKey, buffer, mimeType); err != nil {
		return nil, fmt.Errorf("failed to store file: %w", err)
	}

	// Create upload record
	now := time.Now()
	upload := &Upload{
		ID:           uploadID,
		UserID:       userID,
		Category:     category,
		Status:       StatusStaged,
		OriginalName: filepath.Base(filename),
		MimeType:     mimeType,
		Size:         int64(buffer.Len()),
		StagingKey:   stagingKey,
		CreatedAt:    now,
		ExpiresAt:    now.Add(StagingTTL),
	}

	if err := s.repo.Create(ctx, upload); err != nil {
		// Cleanup staging file on DB error
		_ = s.stagingStorage.Delete(ctx, stagingKey)
		return nil, fmt.Errorf("failed to create upload record: %w", err)
	}

	return upload, nil
}

// Commit moves a staged file to permanent storage
func (s *Service) Commit(ctx context.Context, uploadID uuid.UUID, userID uuid.UUID) (*Upload, error) {
	upload, err := s.repo.GetByID(ctx, uploadID)
	if err != nil || upload == nil {
		return nil, ErrUploadNotFound
	}

	// Check ownership
	if upload.UserID != userID {
		return nil, ErrNotUploadOwner
	}

	// Check status
	if !upload.IsStaged() {
		return nil, ErrAlreadyCommitted
	}

	// Check expiration
	if upload.IsExpired() {
		return nil, ErrUploadExpired
	}

	// Get file from staging
	reader, err := s.stagingStorage.Get(ctx, upload.StagingKey)
	if err != nil {
		upload.Status = StatusFailed
		upload.ErrorMessage = "staging file not found"
		_ = s.repo.Update(ctx, upload)
		return nil, ErrUploadNotFound
	}
	defer reader.Close()

	// Generate permanent key
	ext := storage.GetExtensionForMime(upload.MimeType)
	permanentKey := fmt.Sprintf("%s/%s/%s%s", upload.Category, userID.String(), uploadID.String(), ext)

	// Determine target storage (cloud if available, otherwise staging becomes permanent)
	targetStorage := s.cloudStorage
	if targetStorage == nil {
		targetStorage = s.stagingStorage
	}

	// Store in permanent storage
	// Read all data first since we can't seek
	data, err := io.ReadAll(reader)
	if err != nil {
		upload.Status = StatusFailed
		upload.ErrorMessage = "failed to read staging file"
		_ = s.repo.Update(ctx, upload)
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if err := targetStorage.Put(ctx, permanentKey, NewBytesReader(data), upload.MimeType); err != nil {
		upload.Status = StatusFailed
		upload.ErrorMessage = "failed to store in permanent storage"
		_ = s.repo.Update(ctx, upload)
		return nil, fmt.Errorf("failed to commit file: %w", err)
	}

	// Update upload record
	now := time.Now()
	upload.Status = StatusCommitted
	upload.PermanentKey = permanentKey
	upload.PermanentURL = targetStorage.GetURL(permanentKey)
	upload.CommittedAt = &now

	if err := s.repo.Update(ctx, upload); err != nil {
		return nil, fmt.Errorf("failed to update upload record: %w", err)
	}

	// Cleanup staging file (async, don't fail if it doesn't work)
	go func() {
		_ = s.stagingStorage.Delete(context.Background(), upload.StagingKey)
	}()

	return upload, nil
}

// GetByID returns upload by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Upload, error) {
	upload, err := s.repo.GetByID(ctx, id)
	if err != nil || upload == nil {
		return nil, ErrUploadNotFound
	}
	return upload, nil
}

// Delete removes an upload
func (s *Service) Delete(ctx context.Context, uploadID uuid.UUID, userID uuid.UUID) error {
	upload, err := s.repo.GetByID(ctx, uploadID)
	if err != nil || upload == nil {
		return ErrUploadNotFound
	}

	if upload.UserID != userID {
		return ErrNotUploadOwner
	}

	// Delete from storage
	if upload.IsStaged() && upload.StagingKey != "" {
		_ = s.stagingStorage.Delete(ctx, upload.StagingKey)
	}
	if upload.IsCommitted() && upload.PermanentKey != "" {
		targetStorage := s.cloudStorage
		if targetStorage == nil {
			targetStorage = s.stagingStorage
		}
		_ = targetStorage.Delete(ctx, upload.PermanentKey)
	}

	return s.repo.Delete(ctx, uploadID)
}

// ListByUser returns user's uploads
func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID, category Category) ([]*Upload, error) {
	return s.repo.ListByUser(ctx, userID, category)
}

// CleanupExpired removes expired staged files
func (s *Service) CleanupExpired(ctx context.Context) (int, error) {
	// Get expired uploads
	expired, err := s.repo.ListExpired(ctx, time.Now())
	if err != nil {
		return 0, err
	}

	// Delete staging files
	for _, upload := range expired {
		if upload.StagingKey != "" {
			_ = s.stagingStorage.Delete(ctx, upload.StagingKey)
		}
	}

	// Delete from DB
	return s.repo.DeleteExpired(ctx, time.Now())
}

// GetStagingURL returns the staging URL for an upload
func (s *Service) GetStagingURL(upload *Upload) string {
	return upload.GetURL(s.stagingBaseURL)
}

// BytesReader is a simple bytes.Reader wrapper that implements io.Reader
type bytesReader struct {
	data []byte
	pos  int
}

func NewBytesReader(data []byte) io.Reader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
