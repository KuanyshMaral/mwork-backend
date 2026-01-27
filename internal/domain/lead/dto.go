package lead

import (
	"time"

	"github.com/google/uuid"
)

// CreateLeadRequest for submitting employer lead
type CreateLeadRequest struct {
	// Contact
	ContactName     string `json:"contact_name" validate:"required,min=2,max=255"`
	ContactEmail    string `json:"contact_email" validate:"required,email"`
	ContactPhone    string `json:"contact_phone" validate:"required,min=10,max=20"`
	ContactPosition string `json:"contact_position,omitempty"`

	// Company
	CompanyName    string `json:"company_name" validate:"required,min=2,max=255"`
	BinIIN         string `json:"bin_iin,omitempty" validate:"omitempty,len=12"`
	OrgType        string `json:"org_type,omitempty" validate:"omitempty,oneof=ip too ao agency other"`
	Website        string `json:"website,omitempty"`
	Industry       string `json:"industry,omitempty"`
	EmployeesCount string `json:"employees_count,omitempty" validate:"omitempty,oneof=1-10 11-50 51-200 200+"`

	// Details
	UseCase    string `json:"use_case,omitempty"`
	HowFoundUs string `json:"how_found_us,omitempty"`

	// UTM (from query params)
	UTMSource   string `json:"utm_source,omitempty"`
	UTMMedium   string `json:"utm_medium,omitempty"`
	UTMCampaign string `json:"utm_campaign,omitempty"`
}

// UpdateStatusRequest for updating lead status
type UpdateStatusRequest struct {
	Status          string `json:"status" validate:"required,oneof=new contacted qualified converted rejected lost"`
	Notes           string `json:"notes,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	NextFollowUpAt  string `json:"next_follow_up_at,omitempty"` // RFC3339
}

// AssignRequest for assigning lead to admin
type AssignRequest struct {
	AdminID  string `json:"admin_id" validate:"required,uuid"`
	Priority int    `json:"priority,omitempty" validate:"omitempty,min=0,max=2"`
}

// ConvertRequest for converting lead to employer account
type ConvertRequest struct {
	// Organization details
	LegalName    string `json:"legal_name" validate:"required"`
	BinIIN       string `json:"bin_iin" validate:"required,len=12"`
	OrgType      string `json:"org_type" validate:"required,oneof=ip too ao agency other"`
	LegalAddress string `json:"legal_address,omitempty"`

	// User account
	Password string `json:"password" validate:"required,min=8"`
}

// LeadResponse for API responses
type LeadResponse struct {
	ID             uuid.UUID `json:"id"`
	ContactName    string    `json:"contact_name"`
	ContactEmail   string    `json:"contact_email"`
	ContactPhone   string    `json:"contact_phone"`
	CompanyName    string    `json:"company_name"`
	BinIIN         string    `json:"bin_iin,omitempty"`
	OrgType        string    `json:"org_type,omitempty"`
	Status         string    `json:"status"`
	Priority       int       `json:"priority"`
	Notes          string    `json:"notes,omitempty"`
	AssignedTo     string    `json:"assigned_to,omitempty"`
	NextFollowUpAt string    `json:"next_follow_up_at,omitempty"`
	FollowUpCount  int       `json:"follow_up_count"`
	CreatedAt      string    `json:"created_at"`
}

// ToResponse converts entity to response
func ToResponse(l *EmployerLead) *LeadResponse {
	resp := &LeadResponse{
		ID:            l.ID,
		ContactName:   l.ContactName,
		ContactEmail:  l.ContactEmail,
		ContactPhone:  l.ContactPhone,
		CompanyName:   l.CompanyName,
		Status:        string(l.Status),
		Priority:      l.Priority,
		FollowUpCount: l.FollowUpCount,
		CreatedAt:     l.CreatedAt.Format(time.RFC3339),
	}

	if l.BinIIN.Valid {
		resp.BinIIN = l.BinIIN.String
	}
	if l.OrgType != "" {
		resp.OrgType = string(l.OrgType)
	}
	if l.Notes.Valid {
		resp.Notes = l.Notes.String
	}
	if l.AssignedTo.Valid {
		resp.AssignedTo = l.AssignedTo.UUID.String()
	}
	if l.NextFollowUpAt.Valid {
		resp.NextFollowUpAt = l.NextFollowUpAt.Time.Format(time.RFC3339)
	}

	return resp
}

// LeadSubmittedResponse for public lead submission
type LeadSubmittedResponse struct {
	LeadID  uuid.UUID `json:"lead_id"`
	Message string    `json:"message"`
}
