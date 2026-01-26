package organization

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OrgType represents organization type
type OrgType string

const (
	OrgTypeIP     OrgType = "ip"     // ИП
	OrgTypeTOO    OrgType = "too"    // ТОО
	OrgTypeAO     OrgType = "ao"     // АО
	OrgTypeAgency OrgType = "agency" // Модельное агентство
	OrgTypeOther  OrgType = "other"
)

// VerificationStatus represents verification state
type VerificationStatus string

const (
	VerificationNone     VerificationStatus = "none"
	VerificationPending  VerificationStatus = "pending"
	VerificationInReview VerificationStatus = "in_review"
	VerificationVerified VerificationStatus = "verified"
	VerificationRejected VerificationStatus = "rejected"
)

// Organization represents a legal entity
type Organization struct {
	ID uuid.UUID `db:"id" json:"id"`

	// Legal information
	LegalName string         `db:"legal_name" json:"legal_name"`
	BrandName sql.NullString `db:"brand_name" json:"brand_name,omitempty"`
	BinIIN    string         `db:"bin_iin" json:"bin_iin"`
	OrgType   OrgType        `db:"org_type" json:"org_type"`

	// Address
	LegalAddress  sql.NullString `db:"legal_address" json:"legal_address,omitempty"`
	ActualAddress sql.NullString `db:"actual_address" json:"actual_address,omitempty"`
	City          sql.NullString `db:"city" json:"city,omitempty"`

	// Contacts
	Phone   sql.NullString `db:"phone" json:"phone,omitempty"`
	Email   sql.NullString `db:"email" json:"email,omitempty"`
	Website sql.NullString `db:"website" json:"website,omitempty"`

	// Documents
	RegistrationDocURL sql.NullString  `db:"registration_doc_url" json:"registration_doc_url,omitempty"`
	LicenseDocURL      sql.NullString  `db:"license_doc_url" json:"license_doc_url,omitempty"`
	AdditionalDocs     json.RawMessage `db:"additional_docs" json:"additional_docs,omitempty"`

	// Verification
	VerificationStatus VerificationStatus `db:"verification_status" json:"verification_status"`
	VerificationNotes  sql.NullString     `db:"verification_notes" json:"verification_notes,omitempty"`
	RejectionReason    sql.NullString     `db:"rejection_reason" json:"rejection_reason,omitempty"`
	VerifiedAt         sql.NullTime       `db:"verified_at" json:"verified_at,omitempty"`
	VerifiedBy         uuid.NullUUID      `db:"verified_by" json:"verified_by,omitempty"`

	// Metadata
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// IsVerified returns true if organization is verified
func (o *Organization) IsVerified() bool {
	return o.VerificationStatus == VerificationVerified
}

// IsPending returns true if organization is pending verification
func (o *Organization) IsPending() bool {
	return o.VerificationStatus == VerificationPending || o.VerificationStatus == VerificationInReview
}
