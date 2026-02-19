package attachment

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TargetType defines the business entity that owns the attachment.
// Mirrors the CHECK constraint in the attachments migration.
type TargetType string

const (
	TargetModelPortfolio TargetType = "model_portfolio"
	TargetCastingGallery TargetType = "casting_gallery"
	TargetOrgDocument    TargetType = "org_document"
	TargetChatAttachment TargetType = "chat_attachment"
)

// Attachment links an upload to a business entity (polymorphic 1:N).
// The upload record is the "dumb warehouse"; attachment provides the business label.
type Attachment struct {
	ID         uuid.UUID  `db:"id"          json:"id"`
	UploadID   uuid.UUID  `db:"upload_id"   json:"upload_id"`
	TargetID   uuid.UUID  `db:"target_id"   json:"target_id"`
	TargetType TargetType `db:"target_type" json:"target_type"`
	SortOrder  int        `db:"sort_order"  json:"sort_order"`
	Metadata   Metadata   `db:"metadata"    json:"metadata"`
	CreatedAt  time.Time  `db:"created_at"  json:"created_at"`
}

// ─── Metadata types ──────────────────────────────────────────────────────────
// Each TargetType can carry typed metadata. Using a common envelope avoids
// dealing with raw JSONB in handlers. Add fields per target_type as needed.

// Metadata is the top-level JSONB envelope stored in attachments.metadata.
// The Go service layer populates the appropriate sub-type.
type Metadata struct {
	// For model_portfolio:
	Caption     string `json:"caption,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
	Brand       string `json:"brand,omitempty"`
	Year        int    `json:"year,omitempty"`

	// For org_document:
	DocType string `json:"doc_type,omitempty"` // e.g. "registration", "license"
	Label   string `json:"label,omitempty"`
}

// Value implements driver.Valuer so sqlx can serialize Metadata → JSONB.
func (m Metadata) Value() (driver.Value, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal attachment metadata: %w", err)
	}
	return string(b), nil
}

// Scan implements sql.Scanner so sqlx can deserialize JSONB → Metadata.
func (m *Metadata) Scan(src interface{}) error {
	var b []byte
	switch v := src.(type) {
	case string:
		b = []byte(v)
	case []byte:
		b = v
	case nil:
		return nil
	default:
		return fmt.Errorf("unexpected type for metadata: %T", src)
	}
	return json.Unmarshal(b, m)
}

// AttachmentWithURL combines an attachment with the resolved public file URL.
// Handlers return this enriched type.
type AttachmentWithURL struct {
	Attachment
	URL          string `json:"url"`
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	SizeBytes    int64  `json:"size_bytes"`
}
