package upload

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Status represents the upload lifecycle status
type Status string

const (
	StatusStaged    Status = "staged"    // Uploaded to staging
	StatusCommitted Status = "committed" // Moved to permanent storage
	StatusFailed    Status = "failed"    // Validation/processing failed
	StatusDeleted   Status = "deleted"   // Soft deleted
)

// Category represents the type of upload (DEPRECATED: use Purpose instead)
type Category string

const (
	CategoryAvatar   Category = "avatar"
	CategoryPhoto    Category = "photo"
	CategoryDocument Category = "document"
)

// Purpose represents the intended use of the uploaded file
type Purpose string

const (
	PurposeAvatar       Purpose = "avatar"
	PurposePhoto        Purpose = "photo"
	PurposeDocument     Purpose = "document"
	PurposeCastingCover Purpose = "casting_cover"
	PurposePortfolio    Purpose = "portfolio"
	PurposeChatFile     Purpose = "chat_file"
	PurposeGallery      Purpose = "gallery"
	PurposeVideo        Purpose = "video"
	PurposeAudio        Purpose = "audio"
)

// Upload represents a file upload record
type Upload struct {
	ID       uuid.UUID `db:"id"`
	UserID   uuid.UUID `db:"user_id"`
	Category Category  `db:"category"` // DEPRECATED: use Purpose instead
	Status   Status    `db:"status"`

	// File metadata
	OriginalName string        `db:"original_name"`
	MimeType     string        `db:"mime_type"`
	Size         sql.NullInt64 `db:"size"`

	// Storage keys
	StagingKey   string `db:"staging_key"`   // Key in staging storage
	PermanentKey string `db:"permanent_key"` // Key in permanent (cloud) storage
	PermanentURL string `db:"permanent_url"` // Public URL after commit

	// Processing metadata
	Width  int `db:"width"`  // For images
	Height int `db:"height"` // For images

	// Error tracking
	ErrorMessage string `db:"error_message"`

	// New fields for extended functionality
	Purpose  string        `db:"purpose"`  // Replaces Category eventually
	BatchID  uuid.NullUUID `db:"batch_id"` // For batch uploads
	Metadata []byte        `db:"metadata"` // Custom metadata (JSONB stored as bytes)

	// Timestamps
	CreatedAt   time.Time  `db:"created_at"`
	CommittedAt *time.Time `db:"committed_at"`
	ExpiresAt   time.Time  `db:"expires_at"` // For staged files (auto-cleanup)
}

// IsStaged returns true if file is in staging
func (u *Upload) IsStaged() bool {
	return u.Status == StatusStaged
}

// IsCommitted returns true if file is permanently stored
func (u *Upload) IsCommitted() bool {
	return u.Status == StatusCommitted
}

// IsExpired returns true if staged file has expired
func (u *Upload) IsExpired() bool {
	return u.Status == StatusStaged && time.Now().After(u.ExpiresAt)
}

// GetURL returns the appropriate URL based on status
func (u *Upload) GetURL(stagingBaseURL string) string {
	if u.IsCommitted() && u.PermanentURL != "" {
		return u.PermanentURL
	}
	if u.IsStaged() && u.StagingKey != "" {
		return stagingBaseURL + "/" + u.StagingKey
	}
	return ""
}

func (u *Upload) SizeValue() int64 {
	if !u.Size.Valid {
		return 0
	}
	return u.Size.Int64
}
