package upload

import (
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
	Size         int64     `json:"size"`
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
		Size:         u.Size,
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
