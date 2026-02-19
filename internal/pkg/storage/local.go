package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage implements Storage interface for local file system.
type LocalStorage struct {
	basePath string // Absolute path to root storage directory on disk
	baseURL  string // Public URL prefix (e.g. "http://localhost:8080/static")
}

// NewLocalStorage creates a new local storage instance and ensures the root directory exists.
func NewLocalStorage(basePath, baseURL string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory %q: %w", basePath, err)
	}
	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}, nil
}

// Save stores a file at the given logical path (e.g. "{author_id}/{uuid}.jpg").
// It creates any missing intermediate directories automatically.
func (s *LocalStorage) Save(ctx context.Context, filePath string, reader io.Reader, contentType string) error {
	fullPath := filepath.Join(s.basePath, filepath.FromSlash(filePath))

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %q: %w", filePath, err)
	}

	// Create the destination file
	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", filePath, err)
	}
	defer f.Close()

	// Stream content to disk
	if _, err := io.Copy(f, reader); err != nil {
		// Best-effort cleanup: remove partially written file
		_ = os.Remove(fullPath)
		return fmt.Errorf("failed to write file %q: %w", filePath, err)
	}

	return nil
}

// Delete removes a file by its logical path.
// Returns nil if the file does not exist (idempotent).
func (s *LocalStorage) Delete(ctx context.Context, filePath string) error {
	fullPath := filepath.Join(s.basePath, filepath.FromSlash(filePath))
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already gone — treat as success
		}
		return fmt.Errorf("failed to delete file %q: %w", filePath, err)
	}
	return nil
}

// GetURL returns the public-facing URL for a file given its logical path.
func (s *LocalStorage) GetURL(filePath string) string {
	return s.baseURL + "/" + filePath
}
