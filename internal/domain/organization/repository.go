package organization

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines organization data access
type Repository interface {
	Create(ctx context.Context, org *Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*Organization, error)
	GetByBIN(ctx context.Context, bin string) (*Organization, error)
	Update(ctx context.Context, org *Organization) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, status *VerificationStatus, limit, offset int) ([]*Organization, int, error)
	UpdateVerification(ctx context.Context, id uuid.UUID, status VerificationStatus, notes, reason string, verifiedBy uuid.UUID) error
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates organization repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, org *Organization) error {
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

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
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

func (r *repository) GetByBIN(ctx context.Context, bin string) (*Organization, error) {
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

func (r *repository) Update(ctx context.Context, org *Organization) error {
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

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM organizations WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *repository) List(ctx context.Context, status *VerificationStatus, limit, offset int) ([]*Organization, int, error) {
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

func (r *repository) UpdateVerification(ctx context.Context, id uuid.UUID, status VerificationStatus, notes, reason string, verifiedBy uuid.UUID) error {
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
