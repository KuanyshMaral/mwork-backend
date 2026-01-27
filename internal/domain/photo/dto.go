package photo

import (
	"time"

	"github.com/google/uuid"
)

// PresignRequest for POST /uploads/presign
type PresignRequest struct {
	Filename    string `json:"filename" validate:"required,min=1,max=255"`
	ContentType string `json:"content_type" validate:"required"`
	Size        int64  `json:"size" validate:"required,gt=0"`
}

// PresignResponse for presigned URL
type PresignResponse struct {
	UploadURL string    `json:"upload_url"` // PUT to this URL
	Key       string    `json:"key"`        // Save this for confirmation
	PublicURL string    `json:"public_url"` // Final URL after upload
	ExpiresAt time.Time `json:"expires_at"`
}

// ConfirmUploadRequest for POST /photos
type ConfirmUploadRequest struct {
	Key          string `json:"key" validate:"required"`
	OriginalName string `json:"original_name" validate:"required,max=255"`
}

// SetAvatarRequest for PATCH /photos/{id}/avatar
type SetAvatarRequest struct {
	IsAvatar bool `json:"is_avatar"`
}

// ReorderRequest for PATCH /photos/reorder
type ReorderRequest struct {
	PhotoIDs []uuid.UUID `json:"photo_ids" validate:"required,min=1"`
}

// PhotoResponse represents photo in API response
type PhotoResponse struct {
	ID           uuid.UUID `json:"id"`
	ProfileID    uuid.UUID `json:"profile_id"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	IsAvatar     bool      `json:"is_avatar"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    string    `json:"created_at"`
}

// PhotoResponseFromEntity converts entity to response DTO
func PhotoResponseFromEntity(p *Photo) *PhotoResponse {
	return &PhotoResponse{
		ID:           p.ID,
		ProfileID:    p.ProfileID,
		URL:          p.URL,
		ThumbnailURL: p.GetThumbnailURL(300), // Default thumbnail size
		OriginalName: p.OriginalName,
		MimeType:     p.MimeType,
		SizeBytes:    p.SizeBytes,
		IsAvatar:     p.IsAvatar,
		SortOrder:    p.SortOrder,
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
	}
}
