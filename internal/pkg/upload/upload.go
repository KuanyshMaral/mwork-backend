package upload

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Config for R2 upload service
type Config struct {
	AccountID       string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
	PublicURL       string // CDN URL prefix
}

// Service handles file uploads to Cloudflare R2
type Service struct {
	client    *s3.Client
	presign   *s3.PresignClient
	bucket    string
	publicURL string
}

// NewService creates upload service
// Returns nil if config is incomplete (uploads disabled)
func NewService(cfg *Config) *Service {
	if cfg.AccountID == "" || cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" || cfg.BucketName == "" {
		log.Warn().Msg("R2 config incomplete, file uploads disabled")
		return nil
	}

	// R2 endpoint
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	// Create S3-compatible client for R2
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: endpoint,
		}, nil
	})

	r2Config, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.AccessKeySecret,
			"",
		)),
		config.WithRegion("auto"),
		config.WithEndpointResolverWithOptions(r2Resolver),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create R2 client config")
		return nil
	}

	client := s3.NewFromConfig(r2Config)
	presignClient := s3.NewPresignClient(client)

	publicURL := cfg.PublicURL
	if publicURL == "" {
		// Default R2.dev URL (works if public access enabled)
		publicURL = fmt.Sprintf("https://pub-%s.r2.dev", cfg.AccountID)
	}

	log.Info().Str("bucket", cfg.BucketName).Str("public_url", publicURL).Msg("R2 upload service initialized")

	return &Service{
		client:    client,
		presign:   presignClient,
		bucket:    cfg.BucketName,
		publicURL: strings.TrimSuffix(publicURL, "/"),
	}
}

// PresignResult contains presigned URL data
type PresignResult struct {
	UploadURL string    `json:"upload_url"` // PUT URL for upload
	Key       string    `json:"key"`        // Object key in R2
	PublicURL string    `json:"public_url"` // CDN URL after upload
	ExpiresAt time.Time `json:"expires_at"` // URL expiration
}

// AllowedImageTypes for validation
var AllowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

// MaxFileSize in bytes (10MB)
const MaxFileSize = 10 * 1024 * 1024

// GeneratePresignedURL creates a presigned PUT URL for direct upload
func (s *Service) GeneratePresignedURL(ctx context.Context, userID uuid.UUID, filename string, contentType string, size int64) (*PresignResult, error) {
	if s == nil {
		return nil, fmt.Errorf("upload service not configured")
	}

	// Validate content type
	if !AllowedImageTypes[contentType] {
		return nil, fmt.Errorf("invalid file type: %s (allowed: jpeg, png, webp, gif)", contentType)
	}

	// Validate size
	if size > MaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d MB)", size, MaxFileSize/1024/1024)
	}

	// Generate unique key: photos/2025/01/{user_uuid}/{random}.{ext}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".jpg"
	}
	key := fmt.Sprintf("photos/%s/%s/%s%s",
		time.Now().Format("2006/01"),
		userID.String(),
		uuid.New().String(),
		ext,
	)

	// Create presigned PUT URL (valid 15 min)
	presignedReq, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 15 * time.Minute
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return &PresignResult{
		UploadURL: presignedReq.URL,
		Key:       key,
		PublicURL: fmt.Sprintf("%s/%s", s.publicURL, key),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}, nil
}

// FileMetadata contains verified file info
type FileMetadata struct {
	Key         string
	Size        int64
	ContentType string
}

// VerifyUpload checks if file was uploaded to R2
func (s *Service) VerifyUpload(ctx context.Context, key string) (*FileMetadata, error) {
	if s == nil {
		return nil, fmt.Errorf("upload service not configured")
	}

	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("file not found in storage: %w", err)
	}

	return &FileMetadata{
		Key:         key,
		Size:        aws.ToInt64(head.ContentLength),
		ContentType: aws.ToString(head.ContentType),
	}, nil
}

// DeleteObject removes file from R2
func (s *Service) DeleteObject(ctx context.Context, key string) error {
	if s == nil {
		return nil
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// GetPublicURL returns CDN URL for a key
func (s *Service) GetPublicURL(key string) string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s", s.publicURL, key)
}

// IsConfigured returns true if service is ready
func (s *Service) IsConfigured() bool {
	return s != nil
}
