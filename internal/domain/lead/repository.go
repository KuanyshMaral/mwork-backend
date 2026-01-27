package lead

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines lead data access
type Repository interface {
	Create(ctx context.Context, lead *EmployerLead) error
	GetByID(ctx context.Context, id uuid.UUID) (*EmployerLead, error)
	GetByEmail(ctx context.Context, email string) (*EmployerLead, error)
	Update(ctx context.Context, lead *EmployerLead) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, status *Status, limit, offset int) ([]*EmployerLead, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status, notes, reason string) error
	MarkContacted(ctx context.Context, id uuid.UUID) error
	SetFollowUp(ctx context.Context, id uuid.UUID, followUpAt sql.NullTime) error
	Assign(ctx context.Context, id uuid.UUID, adminID uuid.UUID, priority int) error
	MarkConverted(ctx context.Context, id uuid.UUID, userID, orgID uuid.UUID) error
	CountByStatus(ctx context.Context) (map[Status]int, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates lead repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, lead *EmployerLead) error {
	query := `
		INSERT INTO employer_leads (
			id, contact_name, contact_email, contact_phone, contact_position,
			company_name, bin_iin, org_type, website, industry, employees_count,
			use_case, how_found_us, status, source,
			utm_source, utm_medium, utm_campaign, referrer_url,
			ip_address, user_agent, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19,
			$20, $21, $22, $23
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		lead.ID, lead.ContactName, lead.ContactEmail, lead.ContactPhone, lead.ContactPosition,
		lead.CompanyName, lead.BinIIN, lead.OrgType, lead.Website, lead.Industry, lead.EmployeesCount,
		lead.UseCase, lead.HowFoundUs, lead.Status, lead.Source,
		lead.UTMSource, lead.UTMMedium, lead.UTMCampaign, lead.ReferrerURL,
		lead.IPAddress, lead.UserAgent, lead.CreatedAt, lead.UpdatedAt,
	)
	return err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*EmployerLead, error) {
	query := `SELECT * FROM employer_leads WHERE id = $1`
	var lead EmployerLead
	err := r.db.GetContext(ctx, &lead, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &lead, nil
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*EmployerLead, error) {
	query := `SELECT * FROM employer_leads WHERE contact_email = $1 ORDER BY created_at DESC LIMIT 1`
	var lead EmployerLead
	err := r.db.GetContext(ctx, &lead, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &lead, nil
}

func (r *repository) Update(ctx context.Context, lead *EmployerLead) error {
	query := `
		UPDATE employer_leads SET
			contact_name = $2, contact_phone = $3, contact_position = $4,
			company_name = $5, bin_iin = $6, website = $7,
			notes = $8, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		lead.ID, lead.ContactName, lead.ContactPhone, lead.ContactPosition,
		lead.CompanyName, lead.BinIIN, lead.Website,
		lead.Notes,
	)
	return err
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM employer_leads WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *repository) List(ctx context.Context, status *Status, limit, offset int) ([]*EmployerLead, int, error) {
	var args []interface{}
	where := ""
	argIdx := 1

	if status != nil {
		where = " WHERE status = $1"
		args = append(args, *status)
		argIdx++
	}

	// Count
	countQuery := "SELECT COUNT(*) FROM employer_leads" + where
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// List with proper parameter indices
	query := fmt.Sprintf(`
		SELECT * FROM employer_leads %s 
		ORDER BY priority DESC, created_at DESC 
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	var leads []*EmployerLead
	if err := r.db.SelectContext(ctx, &leads, query, args...); err != nil {
		return nil, 0, err
	}

	return leads, total, nil
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status, notes, reason string) error {
	query := `
		UPDATE employer_leads SET
			status = $2, notes = COALESCE($3, notes), rejection_reason = $4, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, notes, reason)
	return err
}

func (r *repository) MarkContacted(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE employer_leads SET
			last_contacted_at = NOW(), follow_up_count = follow_up_count + 1, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *repository) SetFollowUp(ctx context.Context, id uuid.UUID, followUpAt sql.NullTime) error {
	query := `UPDATE employer_leads SET next_follow_up_at = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, followUpAt)
	return err
}

func (r *repository) Assign(ctx context.Context, id uuid.UUID, adminID uuid.UUID, priority int) error {
	query := `UPDATE employer_leads SET assigned_to = $2, priority = $3, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, adminID, priority)
	return err
}

func (r *repository) MarkConverted(ctx context.Context, id uuid.UUID, userID, orgID uuid.UUID) error {
	query := `
		UPDATE employer_leads SET
			status = 'converted', converted_at = NOW(),
			converted_user_id = $2, converted_org_id = $3, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, userID, orgID)
	return err
}

func (r *repository) CountByStatus(ctx context.Context) (map[Status]int, error) {
	query := `SELECT status, COUNT(*) as count FROM employer_leads GROUP BY status`

	type row struct {
		Status Status `db:"status"`
		Count  int    `db:"count"`
	}

	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}

	result := make(map[Status]int)
	for _, r := range rows {
		result[r.Status] = r.Count
	}
	return result, nil
}
