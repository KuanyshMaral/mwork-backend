package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// LocalStorage implements Storage interface for local file system
type LocalStorage struct {
	basePath string
	baseURL  string
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(basePath, baseURL string) (*LocalStorage, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}, nil
}

// Put stores a file locally
func (s *LocalStorage) Put(ctx context.Context, key string, reader io.Reader, contentType string) error {
	fullPath := filepath.Join(s.basePath, key)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy content
	if _, err := io.Copy(file, reader); err != nil {
		os.Remove(fullPath) // Cleanup on error
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Get retrieves a file from local storage
func (s *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, key)
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// Delete removes a file from local storage
func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.basePath, key)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// Exists checks if a file exists in local storage
func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	fullPath := filepath.Join(s.basePath, key)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetURL returns the URL for a locally stored file
func (s *LocalStorage) GetURL(key string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, key)
}

// GetInfo returns file metadata
func (s *LocalStorage) GetInfo(ctx context.Context, key string) (*FileInfo, error) {
	fullPath := filepath.Join(s.basePath, key)
	stat, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	contentType := ""
	if f, err := os.Open(fullPath); err == nil {
		defer f.Close()
		head := make([]byte, 512)
		n, _ := f.Read(head)
		if n > 0 {
			contentType = http.DetectContentType(head[:n])
		}
	}

	return &FileInfo{
		Key:         key,
		Size:        stat.Size(),
		ContentType: contentType,
		URL:         s.GetURL(key),
	}, nil
}

// CleanupExpired removes files older than the given duration
func (s *LocalStorage) CleanupExpired(ctx context.Context, maxAge time.Duration) (int, error) {
	count := 0
	cutoff := time.Now().Add(-maxAge)

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err == nil {
				count++
			}
		}
		return nil
	})

	return count, err
}
