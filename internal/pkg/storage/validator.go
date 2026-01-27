package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	ErrFileTooLarge    = errors.New("file exceeds maximum size")
	ErrInvalidMimeType = errors.New("file type not allowed")
	ErrEmptyFile       = errors.New("file is empty")
)

// ValidateFile validates file size and MIME type for a given category
func ValidateFile(reader io.Reader, category string, maxSize int64) ([]byte, string, error) {
	// Read file into buffer (limited to maxSize + 1 to detect oversized files)
	limitedReader := io.LimitReader(reader, maxSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	// Check if file is empty
	if len(data) == 0 {
		return nil, "", ErrEmptyFile
	}

	// Check size
	if int64(len(data)) > maxSize {
		return nil, "", ErrFileTooLarge
	}

	// Detect MIME type from content (magic bytes)
	mimeType := http.DetectContentType(data)
	// Clean up MIME type (e.g., "image/jpeg; charset=utf-8" -> "image/jpeg")
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	// Validate MIME type for category
	allowedTypes, ok := AllowedMimeTypes[category]
	if !ok {
		return nil, "", fmt.Errorf("unknown category: %s", category)
	}

	allowed := false
	for _, t := range allowedTypes {
		if t == mimeType {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, "", ErrInvalidMimeType
	}

	return data, mimeType, nil
}

// ValidateAndBuffer reads, validates, and returns a buffer for storage
func ValidateAndBuffer(reader io.Reader, category string) (*bytes.Buffer, string, error) {
	maxSize, ok := MaxFileSizes[category]
	if !ok {
		maxSize = 10 * 1024 * 1024 // Default 10 MB
	}

	data, mimeType, err := ValidateFile(reader, category, maxSize)
	if err != nil {
		return nil, "", err
	}

	return bytes.NewBuffer(data), mimeType, nil
}

// GetExtensionForMime returns the file extension for a MIME type
func GetExtensionForMime(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}
