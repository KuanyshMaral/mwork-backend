package organization

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines organization data access
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates organization repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, org *Organization) error {
	query := `
		INSERT INTO organizations (
			id, legal_name, brand_name, bin_iin, org_type,
			legal_address, actual_address, city,
			phone, email, website,
			verification_status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		org.ID, org.LegalName, org.BrandName, org.BinIIN, org.OrgType,
		org.LegalAddress, org.ActualAddress, org.City,
		org.Phone, org.Email, org.Website,
		org.VerificationStatus, org.CreatedAt, org.UpdatedAt,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	query := `SELECT * FROM organizations WHERE id = $1`
	var org Organization
	err := r.db.GetContext(ctx, &org, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &org, nil
}

func (r *Repository) GetByBIN(ctx context.Context, bin string) (*Organization, error) {
	query := `SELECT * FROM organizations WHERE bin_iin = $1`
	var org Organization
	err := r.db.GetContext(ctx, &org, query, bin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &org, nil
}

func (r *Repository) Update(ctx context.Context, org *Organization) error {
	query := `
		UPDATE organizations SET
			legal_name = $2, brand_name = $3,
			legal_address = $4, actual_address = $5, city = $6,
			phone = $7, email = $8, website = $9,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		org.ID, org.LegalName, org.BrandName,
		org.LegalAddress, org.ActualAddress, org.City,
		org.Phone, org.Email, org.Website,
	)
	return err
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM organizations WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) List(ctx context.Context, status *VerificationStatus, limit, offset int) ([]*Organization, int, error) {
	var args []interface{}
	where := ""
	argIdx := 1

	if status != nil {
		where = " WHERE verification_status = $1"
		args = append(args, *status)
		argIdx++
	}

	// Count
	countQuery := "SELECT COUNT(*) FROM organizations" + where
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// List
	query := "SELECT * FROM organizations" + where +
		" ORDER BY created_at DESC LIMIT $" + string(rune('0'+argIdx)) + " OFFSET $" + string(rune('0'+argIdx+1))
	args = append(args, limit, offset)

	var orgs []*Organization
	if err := r.db.SelectContext(ctx, &orgs, query, args...); err != nil {
		return nil, 0, err
	}

	return orgs, total, nil
}

func (r *Repository) UpdateVerification(ctx context.Context, id uuid.UUID, status VerificationStatus, notes, reason string, verifiedBy uuid.UUID) error {
	query := `
		UPDATE organizations SET
			verification_status = $2,
			verification_notes = $3,
			rejection_reason = $4,
			verified_at = CASE WHEN $2 = 'verified' THEN NOW() ELSE verified_at END,
			verified_by = CASE WHEN $2 = 'verified' THEN $5 ELSE verified_by END,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, notes, reason, verifiedBy)
	return err
}

// Member methods

func (r *Repository) AddMember(ctx context.Context, member *OrganizationMember) error {
	query := `
		INSERT INTO organization_members (
			id, organization_id, user_id, role, invited_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		member.ID, member.OrganizationID, member.UserID, member.Role,
		member.InvitedBy, member.CreatedAt, member.UpdatedAt,
	)
	return err
}

func (r *Repository) GetMembers(ctx context.Context, orgID uuid.UUID) ([]*OrganizationMember, error) {
	query := `SELECT * FROM organization_members WHERE organization_id = $1 ORDER BY created_at`
	var members []*OrganizationMember
	if err := r.db.SelectContext(ctx, &members, query, orgID); err != nil {
		return nil, err
	}
	return members, nil
}

func (r *Repository) GetMemberByID(ctx context.Context, memberID uuid.UUID) (*OrganizationMember, error) {
	query := `SELECT * FROM organization_members WHERE id = $1`
	var member OrganizationMember
	err := r.db.GetContext(ctx, &member, query, memberID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &member, nil
}

func (r *Repository) GetMemberByUserID(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (*OrganizationMember, error) {
	query := `SELECT * FROM organization_members WHERE organization_id = $1 AND user_id = $2`
	var member OrganizationMember
	err := r.db.GetContext(ctx, &member, query, orgID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &member, nil
}

func (r *Repository) UpdateMemberRole(ctx context.Context, memberID uuid.UUID, role MemberRole) error {
	query := `UPDATE organization_members SET role = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, memberID, role)
	return err
}

func (r *Repository) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	query := `DELETE FROM organization_members WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, memberID)
	return err
}

// Follower methods

func (r *Repository) AddFollower(ctx context.Context, follower *AgencyFollower) error {
	query := `
		INSERT INTO agency_followers (id, organization_id, follower_user_id, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.ExecContext(ctx, query,
		follower.ID, follower.OrganizationID, follower.FollowerUserID, follower.CreatedAt,
	)
	return err
}

func (r *Repository) RemoveFollower(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM agency_followers WHERE organization_id = $1 AND follower_user_id = $2`
	_, err := r.db.ExecContext(ctx, query, orgID, userID)
	return err
}

func (r *Repository) GetFollowers(ctx context.Context, orgID uuid.UUID) ([]*AgencyFollower, error) {
	query := `SELECT * FROM agency_followers WHERE organization_id = $1 ORDER BY created_at DESC`
	var followers []*AgencyFollower
	if err := r.db.SelectContext(ctx, &followers, query, orgID); err != nil {
		return nil, err
	}
	return followers, nil
}

func (r *Repository) IsFollowing(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM agency_followers WHERE organization_id = $1 AND follower_user_id = $2)`
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, orgID, userID)
	return exists, err
}

func (r *Repository) GetFollowerUserIDs(ctx context.Context, orgID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT follower_user_id FROM agency_followers WHERE organization_id = $1`
	var userIDs []uuid.UUID
	if err := r.db.SelectContext(ctx, &userIDs, query, orgID); err != nil {
		return nil, err
	}
	return userIDs, nil
}
