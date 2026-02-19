package upload

import (
	"time"

	"github.com/google/uuid"
)

// Upload represents a stored file in the "dumb warehouse".
// It knows nothing about business logic — no purposes, no categories.
type Upload struct {
	ID           uuid.UUID `db:"id"            json:"id"`
	AuthorID     uuid.UUID `db:"author_id"     json:"author_id"`
	FilePath     string    `db:"file_path"     json:"file_path"` // Logical path: "{author_id}/{uuid}.ext"
	OriginalName string    `db:"original_name" json:"original_name"`
	MimeType     string    `db:"mime_type"     json:"mime_type"`
	SizeBytes    int64     `db:"size_bytes"    json:"size_bytes"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
}
