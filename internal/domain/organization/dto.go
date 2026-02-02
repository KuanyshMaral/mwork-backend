package organization

import (
	"github.com/google/uuid"
)

// CreateRequest for creating organization
type CreateOrganizationRequest struct {
	LegalName     string  `json:"legal_name" validate:"required,min=2,max=255"`
	BrandName     string  `json:"brand_name,omitempty"`
	BinIIN        string  `json:"bin_iin" validate:"required,len=12"`
	OrgType       OrgType `json:"org_type" validate:"required"`
	LegalAddress  string  `json:"legal_address,omitempty"`
	ActualAddress string  `json:"actual_address,omitempty"`
	City          string  `json:"city,omitempty"`
	Phone         string  `json:"phone,omitempty"`
	Email         string  `json:"email,omitempty" validate:"omitempty,email"`
	Website       string  `json:"website,omitempty"`
}

// For backwards compatibility
type CreateRequest = CreateOrganizationRequest

// UpdateRequest for updating organization
type UpdateRequest struct {
	LegalName     *string `json:"legal_name,omitempty" validate:"omitempty,min=2,max=255"`
	BrandName     *string `json:"brand_name,omitempty"`
	LegalAddress  *string `json:"legal_address,omitempty"`
	ActualAddress *string `json:"actual_address,omitempty"`
	City          *string `json:"city,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	Email         *string `json:"email,omitempty" validate:"omitempty,email"`
	Website       *string `json:"website,omitempty"`
}

// VerifyRequest for verifying/rejecting organization
type VerifyRequest struct {
	Status          string `json:"status" validate:"required,oneof=verified rejected"`
	Notes           string `json:"notes,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// UploadDocRequest for document upload
type UploadDocRequest struct {
	DocType string `json:"doc_type" validate:"required,oneof=registration license additional"`
	URL     string `json:"url" validate:"required,url"`
}

// Response for organization data
type Response struct {
	ID                 uuid.UUID `json:"id"`
	LegalName          string    `json:"legal_name"`
	BrandName          string    `json:"brand_name,omitempty"`
	BinIIN             string    `json:"bin_iin"`
	OrgType            string    `json:"org_type"`
	City               string    `json:"city,omitempty"`
	Phone              string    `json:"phone,omitempty"`
	Email              string    `json:"email,omitempty"`
	Website            string    `json:"website,omitempty"`
	VerificationStatus string    `json:"verification_status"`
	IsVerified         bool      `json:"is_verified"`
	CreatedAt          string    `json:"created_at"`
}

// ToResponse converts entity to response
func ToResponse(org *Organization) *Response {
	resp := &Response{
		ID:                 org.ID,
		LegalName:          org.LegalName,
		BinIIN:             org.BinIIN,
		OrgType:            string(org.OrgType),
		VerificationStatus: string(org.VerificationStatus),
		IsVerified:         org.IsVerified(),
		CreatedAt:          org.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if org.BrandName.Valid {
		resp.BrandName = org.BrandName.String
	}
	if org.City.Valid {
		resp.City = org.City.String
	}
	if org.Phone.Valid {
		resp.Phone = org.Phone.String
	}
	if org.Email.Valid {
		resp.Email = org.Email.String
	}
	if org.Website.Valid {
		resp.Website = org.Website.String
	}

	return resp
}

// AddMemberRequest for adding organization member
type AddMemberRequest struct {
	UserID uuid.UUID  `json:"user_id" validate:"required"`
	Role   MemberRole `json:"role" validate:"required"`
}

// UpdateMemberRoleRequest for updating member role
type UpdateMemberRoleRequest struct {
	Role MemberRole `json:"role" validate:"required"`
}

// MemberResponse for organization member data
type MemberResponse struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Role      MemberRole `json:"role"`
	InvitedBy *uuid.UUID `json:"invited_by,omitempty"`
	CreatedAt string     `json:"created_at"`
}

// ToMemberResponse converts OrganizationMember to MemberResponse
func ToMemberResponse(member *OrganizationMember) *MemberResponse {
	resp := &MemberResponse{
		ID:        member.ID,
		UserID:    member.UserID,
		Role:      member.Role,
		CreatedAt: member.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if member.InvitedBy.Valid {
		resp.InvitedBy = &member.InvitedBy.UUID
	}
	return resp
}

// FollowerResponse for follower data
type FollowerResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt string    `json:"created_at"`
}

// ToFollowerResponse converts AgencyFollower to FollowerResponse
func ToFollowerResponse(follower *AgencyFollower) *FollowerResponse {
	return &FollowerResponse{
		UserID:    follower.FollowerUserID,
		CreatedAt: follower.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
