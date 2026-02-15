package storage

import (
	"context"
	"fmt"
	"time"
)

// UploadStorageAdapter adapts Storage to the upload handler's extended needs.
// For local storage, presigned URLs are represented as direct backend URLs.
type UploadStorageAdapter struct {
	storage Storage
}

func NewUploadStorageAdapter(st Storage) *UploadStorageAdapter {
	return &UploadStorageAdapter{storage: st}
}

func (a *UploadStorageAdapter) GeneratePresignedPutURL(ctx context.Context, key string, expires time.Duration, contentType string) (string, error) {
	_ = ctx
	_ = expires
	_ = contentType
	return a.storage.GetURL(key), nil
}

func (a *UploadStorageAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.storage.Exists(ctx, key)
}

func (a *UploadStorageAdapter) Move(ctx context.Context, srcKey, dstKey string) error {
	reader, err := a.storage.Get(ctx, srcKey)
	if err != nil {
		return fmt.Errorf("failed to open source object: %w", err)
	}
	defer reader.Close()

	info, err := a.storage.GetInfo(ctx, srcKey)
	if err != nil {
		return fmt.Errorf("failed to read source metadata: %w", err)
	}

	if err := a.storage.Put(ctx, dstKey, reader, info.ContentType); err != nil {
		return fmt.Errorf("failed to write destination object: %w", err)
	}

	if err := a.storage.Delete(ctx, srcKey); err != nil {
		return fmt.Errorf("failed to delete source object: %w", err)
	}

	return nil
}

func (a *UploadStorageAdapter) GetURL(key string) string {
	return a.storage.GetURL(key)
}
