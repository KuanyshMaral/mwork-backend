package upload

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// StageRequest for POST /uploads/stage
type StageRequest struct {
	Category string `json:"category" validate:"required,oneof=avatar photo document"`
}

// UploadResponse represents upload in API response
type UploadResponse struct {
	ID           uuid.UUID `json:"id"`
	Category     string    `json:"category"`
	Status       string    `json:"status"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	Size         *int64    `json:"size,omitempty"`
	URL          string    `json:"url"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	CreatedAt    string    `json:"created_at"`
	ExpiresAt    *string   `json:"expires_at,omitempty"` // Only for staged files
}

// UploadResponseFromEntity converts entity to response
func UploadResponseFromEntity(u *Upload, stagingBaseURL string) *UploadResponse {
	resp := &UploadResponse{
		ID:           u.ID,
		Category:     string(u.Category),
		Status:       string(u.Status),
		OriginalName: u.OriginalName,
		MimeType:     u.MimeType,
		Size:         nullInt64Ptr(u.Size),
		URL:          u.GetURL(stagingBaseURL),
		Width:        u.Width,
		Height:       u.Height,
		CreatedAt:    u.CreatedAt.Format(time.RFC3339),
	}

	if u.IsStaged() {
		exp := u.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &exp
	}

	return resp
}

type InitRequestDoc struct {
	FileName    string            `json:"file_name" validate:"required" example:"avatar.jpg"`
	ContentType string            `json:"content_type" validate:"required" example:"image/jpeg"`
	FileSize    int64             `json:"file_size" validate:"required" example:"123456"`
	Purpose     string            `json:"purpose" validate:"required" enums:"avatar,portfolio,casting_cover,chat_file,photo,document" example:"avatar"`
	BatchID     string            `json:"batch_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type InitResponseDoc struct {
	UploadID  string            `json:"upload_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UploadURL string            `json:"upload_url,omitempty" example:"https://example.com/presigned-put-url"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt string            `json:"expires_at" example:"2026-01-01T12:00:00Z"`
	Purpose   string            `json:"purpose" enums:"avatar,portfolio,casting_cover,chat_file,photo,document" example:"avatar"`
}

type ConfirmRequestDoc struct {
	UploadID string `json:"upload_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type ConfirmResponseDoc struct {
	UploadID     string `json:"upload_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status       string `json:"status" example:"committed"`
	PermanentURL string `json:"permanent_url" example:"https://cdn.example.com/uploads/final/x.jpg"`
	ContentType  string `json:"content_type" example:"image/jpeg"`
	FileSize     int64  `json:"file_size" example:"123456"`
	Purpose      string `json:"purpose" enums:"avatar,portfolio,casting_cover,chat_file,photo,document" example:"avatar"`
}

func nullInt64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	vv := v.Int64
	return &vv
}
