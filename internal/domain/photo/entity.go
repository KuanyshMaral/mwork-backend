package photo

import (
	"time"

	"github.com/google/uuid"
)

// Photo represents a photo in model portfolio (metadata only, file in R2)
type Photo struct {
	ID           uuid.UUID `db:"id" json:"id"`
	ProfileID    uuid.UUID `db:"profile_id" json:"profile_id"`
	Key          string    `db:"key" json:"key"` // R2 object key
	URL          string    `db:"url" json:"url"` // Public CDN URL
	OriginalName string    `db:"original_name" json:"original_name"`
	MimeType     string    `db:"mime_type" json:"mime_type"`
	SizeBytes    int64     `db:"size_bytes" json:"size_bytes"`
	IsAvatar     bool      `db:"is_avatar" json:"is_avatar"`
	SortOrder    int       `db:"sort_order" json:"sort_order"`
	Caption      string    `db:"caption" json:"caption"`
	ProjectName  string    `db:"project_name" json:"project_name"`
	Brand        string    `db:"brand" json:"brand"`
	Year         int       `db:"year" json:"year"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// GetThumbnailURL returns resized image URL (via Cloudflare Worker)
func (p *Photo) GetThumbnailURL(width int) string {
	// For future: https://img.mwork.kz/{key}?w={width}
	// For now, return original
	return p.URL
}
