package upload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/pkg/storage"
)

const (
	// MaxUploadSize is the global maximum for any single file upload.
	MaxUploadSize = 50 * 1024 * 1024 // 50 MB
)

// AllowedMimeTypes is a flat global whitelist.
// No per-category complexity — if it's in the list, we accept it.
var AllowedMimeTypes = map[string]bool{
	"image/jpeg":         true,
	"image/png":          true,
	"image/webp":         true,
	"image/gif":          true,
	"video/mp4":          true,
	"video/quicktime":    true,
	"audio/mpeg":         true,
	"audio/wav":          true,
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"text/plain":      true,
	"application/zip": true,
}

// Service handles file upload business logic.
type Service struct {
	repo    Repository
	storage storage.Storage
	baseURL string // Public URL prefix for generating file URLs
}

// NewService creates a new upload service.
func NewService(repo Repository, storage storage.Storage, baseURL string) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		baseURL: baseURL,
	}
}

// Upload saves a file to disk and registers it in the database.
// filePath is the logical path relative to storage root: "{authorID}/{uuid}.ext"
func (s *Service) Upload(ctx context.Context, authorID uuid.UUID, filename string, reader io.Reader) (*Upload, error) {
	// Read file into buffer to get size and detect MIME type
	buf := &bytes.Buffer{}
	size, err := io.Copy(buf, io.LimitReader(reader, MaxUploadSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if size > MaxUploadSize {
		return nil, ErrFileTooLarge
	}

	// Detect MIME type from file content (first 512 bytes)
	mimeType := http.DetectContentType(buf.Bytes())
	// Normalize: http.DetectContentType can return "text/plain; charset=utf-8"
	mimeType, _, _ = mime.ParseMediaType(mimeType)

	if !AllowedMimeTypes[mimeType] {
		return nil, ErrInvalidMime
	}

	// Derive extension from original filename as a hint; fall back to MIME-based extension
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		exts, _ := mime.ExtensionsByType(mimeType)
		if len(exts) > 0 {
			ext = exts[0]
		}
	}

	// Build logical path: "{author_id}/{uuid}.ext"
	fileID := uuid.New()
	filePath := fmt.Sprintf("%s/%s%s", authorID.String(), fileID.String(), ext)

	// Persist to storage backend
	if err := s.storage.Save(ctx, filePath, bytes.NewReader(buf.Bytes()), mimeType); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Write metadata to database
	upload := &Upload{
		ID:           fileID,
		AuthorID:     authorID,
		FilePath:     filePath,
		OriginalName: filepath.Base(filename),
		MimeType:     mimeType,
		SizeBytes:    size,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, upload); err != nil {
		// Best-effort: try to clean up the file if DB write fails
		_ = s.storage.Delete(ctx, filePath)
		return nil, fmt.Errorf("failed to register upload: %w", err)
	}

	return upload, nil
}

// GetByID retrieves file metadata by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Upload, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUploadNotFound
	}
	return u, nil
}

// Delete removes a file from storage and database.
// Only the file's author can delete it.
func (s *Service) Delete(ctx context.Context, id uuid.UUID, callerID uuid.UUID) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUploadNotFound
	}
	if u.AuthorID != callerID {
		return ErrNotOwner
	}

	// Delete from DB first; physical file cleanup is secondary
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete upload record: %w", err)
	}

	// Delete physical file — non-fatal if it fails (cron can clean up later)
	_ = s.storage.Delete(ctx, u.FilePath)

	return nil
}

// ListByAuthor returns all uploads belonging to a user.
func (s *Service) ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]*Upload, error) {
	return s.repo.ListByAuthor(ctx, authorID)
}

// GetURL builds the full public URL for an upload.
func (s *Service) GetURL(u *Upload) string {
	return s.storage.GetURL(u.FilePath)
}
