package storage

import (
	"context"
	"io"
)

// FileInfo represents metadata about a stored file
type FileInfo struct {
	Key         string // Unique identifier/path
	Size        int64
	ContentType string
	URL         string // Public URL if available
}

// Storage defines the interface for file storage backends
type Storage interface {
	// Put stores a file and returns its key
	Put(ctx context.Context, key string, reader io.Reader, contentType string) error

	// Get retrieves a file by key
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes a file by key
	Delete(ctx context.Context, key string) error

	// Exists checks if a file exists
	Exists(ctx context.Context, key string) (bool, error)

	// GetURL returns the public URL for a file
	GetURL(key string) string

	// GetInfo returns file metadata
	GetInfo(ctx context.Context, key string) (*FileInfo, error)
}

// AllowedMimeTypes defines allowed file types per category
var AllowedMimeTypes = map[string][]string{
	"avatar": {
		"image/jpeg",
		"image/png",
		"image/webp",
	},
	"photo": {
		"image/jpeg",
		"image/png",
		"image/webp",
	},
	"document": {
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"text/plain",
	},
	"casting_cover": {
		"image/jpeg",
		"image/png",
		"image/webp",
	},
	"portfolio": {
		"image/jpeg",
		"image/png",
		"image/webp",
	},
	"gallery": {
		"image/jpeg",
		"image/png",
		"image/webp",
	},
	"chat_file": {
		"image/jpeg",
		"image/png",
		"image/webp",
		"image/gif",
		"video/mp4",
		"video/quicktime",
		"audio/mpeg",
		"audio/wav",
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"text/plain",
		"application/zip",
	},
	"video": {
		"video/mp4",
		"video/quicktime",
	},
	"audio": {
		"audio/mpeg",
		"audio/wav",
	},
}

// MaxFileSizes defines max file size per category (bytes)
var MaxFileSizes = map[string]int64{
	"avatar":        5 * 1024 * 1024,   // 5 MB
	"photo":         10 * 1024 * 1024,  // 10 MB
	"document":      20 * 1024 * 1024,  // 20 MB
	"casting_cover": 10 * 1024 * 1024,  // 10 MB
	"portfolio":     10 * 1024 * 1024,  // 10 MB
	"gallery":       10 * 1024 * 1024,  // 10 MB
	"chat_file":     50 * 1024 * 1024,  // 50 MB
	"video":         100 * 1024 * 1024, // 100 MB
	"audio":         20 * 1024 * 1024,  // 20 MB
}

// Config holds storage configuration
type Config struct {
	Type        string // "local", "s3", "r2"
	LocalPath   string // For local storage: path to store files
	LocalURL    string // For local storage: public URL prefix
	S3Endpoint  string // For S3/MinIO: custom endpoint
	S3Region    string // AWS region
	S3Bucket    string // S3 bucket name
	S3AccessKey string // S3 access key
	S3SecretKey string // S3 secret key
}

// New creates a storage instance based on config
func New(cfg Config) (Storage, error) {
	switch cfg.Type {
	case "local":
		return NewLocalStorage(cfg.LocalPath, cfg.LocalURL)
	case "s3", "minio":
		return NewS3Storage(cfg)
	case "r2":
		// R2 uses its own config, caller should use NewR2Storage directly
		return nil, nil
	default:
		return NewLocalStorage(cfg.LocalPath, cfg.LocalURL)
	}
}
