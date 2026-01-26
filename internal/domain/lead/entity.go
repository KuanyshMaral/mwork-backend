package lead

import (
	"database/sql"
	"net"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/organization"
)

// Status represents lead status
type Status string

const (
	StatusNew       Status = "new"
	StatusContacted Status = "contacted"
	StatusQualified Status = "qualified"
	StatusConverted Status = "converted"
	StatusRejected  Status = "rejected"
	StatusLost      Status = "lost"
)

// EmployerLead represents a potential employer lead
type EmployerLead struct {
	ID uuid.UUID `db:"id" json:"id"`

	// Contact person
	ContactName     string         `db:"contact_name" json:"contact_name"`
	ContactEmail    string         `db:"contact_email" json:"contact_email"`
	ContactPhone    string         `db:"contact_phone" json:"contact_phone"`
	ContactPosition sql.NullString `db:"contact_position" json:"contact_position,omitempty"`

	// Company info
	CompanyName    string               `db:"company_name" json:"company_name"`
	BinIIN         sql.NullString       `db:"bin_iin" json:"bin_iin,omitempty"`
	OrgType        organization.OrgType `db:"org_type" json:"org_type,omitempty"`
	Website        sql.NullString       `db:"website" json:"website,omitempty"`
	Industry       sql.NullString       `db:"industry" json:"industry,omitempty"`
	EmployeesCount sql.NullString       `db:"employees_count" json:"employees_count,omitempty"`

	// Application details
	UseCase                  sql.NullString `db:"use_case" json:"use_case,omitempty"`
	ExpectedCastingsPerMonth sql.NullInt32  `db:"expected_castings_per_month" json:"expected_castings_per_month,omitempty"`
	HowFoundUs               sql.NullString `db:"how_found_us" json:"how_found_us,omitempty"`

	// Lead management
	Status     Status         `db:"status" json:"status"`
	Priority   int            `db:"priority" json:"priority"`
	AssignedTo uuid.NullUUID  `db:"assigned_to" json:"assigned_to,omitempty"`
	Notes      sql.NullString `db:"notes" json:"notes,omitempty"`

	// Follow-up
	LastContactedAt sql.NullTime `db:"last_contacted_at" json:"last_contacted_at,omitempty"`
	NextFollowUpAt  sql.NullTime `db:"next_follow_up_at" json:"next_follow_up_at,omitempty"`
	FollowUpCount   int          `db:"follow_up_count" json:"follow_up_count"`

	// Conversion
	ConvertedAt     sql.NullTime   `db:"converted_at" json:"converted_at,omitempty"`
	ConvertedUserID uuid.NullUUID  `db:"converted_user_id" json:"converted_user_id,omitempty"`
	ConvertedOrgID  uuid.NullUUID  `db:"converted_org_id" json:"converted_org_id,omitempty"`
	RejectionReason sql.NullString `db:"rejection_reason" json:"rejection_reason,omitempty"`

	// UTM tracking
	Source      sql.NullString `db:"source" json:"source,omitempty"`
	UTMSource   sql.NullString `db:"utm_source" json:"utm_source,omitempty"`
	UTMMedium   sql.NullString `db:"utm_medium" json:"utm_medium,omitempty"`
	UTMCampaign sql.NullString `db:"utm_campaign" json:"utm_campaign,omitempty"`
	ReferrerURL sql.NullString `db:"referrer_url" json:"referrer_url,omitempty"`

	// Metadata
	IPAddress net.IP         `db:"ip_address" json:"-"`
	UserAgent sql.NullString `db:"user_agent" json:"-"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt time.Time      `db:"updated_at" json:"updated_at"`
}

// IsNew returns true if lead is new
func (l *EmployerLead) IsNew() bool {
	return l.Status == StatusNew
}

// IsConverted returns true if lead was converted
func (l *EmployerLead) IsConverted() bool {
	return l.Status == StatusConverted
}
