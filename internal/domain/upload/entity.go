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

// Category represents the type of upload
type Category string

const (
	CategoryAvatar   Category = "avatar"
	CategoryPhoto    Category = "photo"
	CategoryDocument Category = "document"
)

// Upload represents a file upload record
type Upload struct {
	ID       uuid.UUID `db:"id"`
	UserID   uuid.UUID `db:"user_id"`
	Category Category  `db:"category"`
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
