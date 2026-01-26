package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Storage implements Storage interface for Cloudflare R2
type R2Storage struct {
	client    *s3.Client
	bucket    string
	publicURL string // CDN URL for public access
}

// R2Config holds R2 connection configuration
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
	PublicURL       string // e.g., https://cdn.mwork.kz
}

// NewR2Storage creates a new Cloudflare R2 storage instance
func NewR2Storage(cfg R2Config) (*R2Storage, error) {
	// R2 endpoint format: https://<account_id>.r2.cloudflarestorage.com
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	// Configure AWS SDK for R2
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: endpoint,
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.AccessKeySecret,
			"",
		)),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load R2 config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)

	return &R2Storage{
		client:    client,
		bucket:    cfg.BucketName,
		publicURL: cfg.PublicURL,
	}, nil
}

// Put stores a file in R2
func (s *R2Storage) Put(ctx context.Context, key string, reader io.Reader, contentType string) error {
	// Read all data since S3 SDK needs content length
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        NewBytesReadSeeker(data),
		ContentType: aws.String(contentType),
	}

	_, err = s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to R2: %w", err)
	}

	return nil
}

// Get retrieves a file from R2
func (s *R2Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get from R2: %w", err)
	}

	return result.Body, nil
}

// Delete removes a file from R2
func (s *R2Storage) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	return nil
}

// Exists checks if a file exists in R2
func (s *R2Storage) Exists(ctx context.Context, key string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.HeadObject(ctx, input)
	if err != nil {
		// Check if it's a "not found" error
		return false, nil
	}

	return true, nil
}

// GetURL returns the public URL for an R2 file
func (s *R2Storage) GetURL(key string) string {
	if s.publicURL != "" {
		return fmt.Sprintf("%s/%s", s.publicURL, key)
	}
	// Fallback to direct R2 URL (requires public bucket)
	return fmt.Sprintf("https://%s.r2.dev/%s", s.bucket, key)
}

// GetInfo returns file metadata from R2
func (s *R2Storage) GetInfo(ctx context.Context, key string) (*FileInfo, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info from R2: %w", err)
	}

	contentType := ""
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	return &FileInfo{
		Key:         key,
		Size:        *result.ContentLength,
		ContentType: contentType,
		URL:         s.GetURL(key),
	}, nil
}

// BytesReadSeeker implements io.ReadSeeker for []byte
type bytesReadSeeker struct {
	data []byte
	pos  int64
}

func NewBytesReadSeeker(data []byte) io.ReadSeeker {
	return &bytesReadSeeker{data: data}
}

func (r *bytesReadSeeker) Read(p []byte) (n int, err error) {
	if r.pos >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += int64(n)
	return n, nil
}

func (r *bytesReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = r.pos + offset
	case io.SeekEnd:
		newPos = int64(len(r.data)) + offset
	}
	if newPos < 0 {
		return 0, fmt.Errorf("negative position")
	}
	r.pos = newPos
	return newPos, nil
}
