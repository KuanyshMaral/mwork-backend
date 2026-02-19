package storage

import (
	"context"
	"io"
)

// Storage defines the minimal interface for file storage backends.
// Intentionally simple: Save a file, Delete a file, get its URL.
type Storage interface {
	// Save stores a file at the given path and returns an error on failure.
	Save(ctx context.Context, filePath string, reader io.Reader, contentType string) error

	// Delete removes a file by its path. Returns nil if file doesn't exist.
	Delete(ctx context.Context, filePath string) error

	// GetURL returns the public URL for a file given its logical path.
	GetURL(filePath string) string
}
