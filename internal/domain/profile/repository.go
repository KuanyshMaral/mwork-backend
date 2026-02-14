package profile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Filter represents search filters for models
type Filter struct {
	City      *string
	Gender    *string
	AgeMin    *int
	AgeMax    *int
	HeightMin *float64
	HeightMax *float64
	IsPublic  *bool
	Query     *string
}

// Pagination represents pagination params
type Pagination struct {
	Page  int
	Limit int
}

// ModelRepository defines model profile data access interface
type ModelRepository interface {
	Create(ctx context.Context, profile *ModelProfile) error
	GetByID(ctx context.Context, id uuid.UUID) (*ModelProfile, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*ModelProfile, error)
	Update(ctx context.Context, profile *ModelProfile) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *Filter, pagination *Pagination) ([]*ModelProfile, int, error)
	ListPromoted(ctx context.Context, city *string, limit int) ([]*ModelProfile, error)
	IncrementViewCount(ctx context.Context, id uuid.UUID) error
}

// EmployerRepository defines employer profile data access interface
type EmployerRepository interface {
	Create(ctx context.Context, profile *EmployerProfile) error
	GetByID(ctx context.Context, id uuid.UUID) (*EmployerProfile, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*EmployerProfile, error)
	Update(ctx context.Context, profile *EmployerProfile) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type AdminRepository interface {
	Create(ctx context.Context, profile *AdminProfile) error
	GetByID(ctx context.Context, id uuid.UUID) (*AdminProfile, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*AdminProfile, error)
	Update(ctx context.Context, profile *AdminProfile) error
}

// ---- MODEL REPOSITORY ----

type modelRepository struct{ db *sqlx.DB }

func NewModelRepository(db *sqlx.DB) ModelRepository { return &modelRepository{db: db} }

func (r *modelRepository) Create(ctx context.Context, profile *ModelProfile) error {
	q := `INSERT INTO model_profiles (
		id, user_id, name, bio, description, age, height, weight, gender,
		clothing_size, shoe_size, experience, hourly_rate, city, country,
		languages, categories, skills, barter_accepted, accept_remote_work,
		travel_cities, visibility, profile_views, rating, total_reviews, is_public,
		created_at, updated_at
	) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,
		$10,$11,$12,$13,$14,$15,
		$16,$17,$18,$19,$20,
		$21,$22,$23,$24,$25,$26,
		$27,$28
	)`
	_, err := r.db.ExecContext(ctx, q,
		profile.ID, profile.UserID, profile.Name, profile.Bio, profile.Description,
		profile.Age, profile.Height, profile.Weight, profile.Gender,
		profile.ClothingSize, profile.ShoeSize, profile.Experience, profile.HourlyRate,
		profile.City, profile.Country, profile.Languages, profile.Categories, profile.Skills,
		profile.BarterAccepted, profile.AcceptRemoteWork, profile.TravelCities, profile.Visibility,
		profile.ProfileViews, profile.Rating, profile.TotalReviews, profile.IsPublic,
		profile.CreatedAt, profile.UpdatedAt,
	)
	return err
}

func (r *modelRepository) GetByID(ctx context.Context, id uuid.UUID) (*ModelProfile, error) {
	q := `SELECT id,user_id,name,bio,description,age,height,weight,gender,clothing_size,shoe_size,experience,
	hourly_rate,city,country,languages,categories,skills,barter_accepted,accept_remote_work,travel_cities,
	visibility,profile_views,rating,total_reviews,is_public,created_at,updated_at FROM model_profiles WHERE id=$1`
	var p ModelProfile
	err := r.db.GetContext(ctx, &p, q, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func (r *modelRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*ModelProfile, error) {
	q := `SELECT id,user_id,name,bio,description,age,height,weight,gender,clothing_size,shoe_size,experience,
	hourly_rate,city,country,languages,categories,skills,barter_accepted,accept_remote_work,travel_cities,
	visibility,profile_views,rating,total_reviews,is_public,created_at,updated_at FROM model_profiles WHERE user_id=$1`
	var p ModelProfile
	err := r.db.GetContext(ctx, &p, q, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func (r *modelRepository) Update(ctx context.Context, p *ModelProfile) error {
	q := `UPDATE model_profiles SET
	name=$2,bio=$3,description=$4,age=$5,height=$6,weight=$7,gender=$8,clothing_size=$9,shoe_size=$10,
	experience=$11,hourly_rate=$12,city=$13,country=$14,languages=$15,categories=$16,skills=$17,
	barter_accepted=$18,accept_remote_work=$19,is_public=$20,travel_cities=$21,visibility=$22,updated_at=NOW()
	WHERE id=$1`
	_, err := r.db.ExecContext(ctx, q, p.ID, p.Name, p.Bio, p.Description, p.Age, p.Height, p.Weight, p.Gender, p.ClothingSize, p.ShoeSize, p.Experience, p.HourlyRate, p.City, p.Country, p.Languages, p.Categories, p.Skills, p.BarterAccepted, p.AcceptRemoteWork, p.IsPublic, p.TravelCities, p.Visibility)
	return err
}
func (r *modelRepository) List(ctx context.Context, filter *Filter, pagination *Pagination) ([]*ModelProfile, int, error) {
	conditions := []string{"is_public = true"}
	args := []interface{}{}
	argIndex := 1
	if filter.City != nil && *filter.City != "" {
		conditions = append(conditions, fmt.Sprintf("city ILIKE $%d", argIndex))
		args = append(args, "%"+*filter.City+"%")
		argIndex++
	}
	if filter.Gender != nil && *filter.Gender != "" {
		conditions = append(conditions, fmt.Sprintf("gender=$%d", argIndex))
		args = append(args, *filter.Gender)
		argIndex++
	}
	if filter.AgeMin != nil {
		conditions = append(conditions, fmt.Sprintf("age >= $%d", argIndex))
		args = append(args, *filter.AgeMin)
		argIndex++
	}
	if filter.AgeMax != nil {
		conditions = append(conditions, fmt.Sprintf("age <= $%d", argIndex))
		args = append(args, *filter.AgeMax)
		argIndex++
	}
	if filter.HeightMin != nil {
		conditions = append(conditions, fmt.Sprintf("height >= $%d", argIndex))
		args = append(args, *filter.HeightMin)
		argIndex++
	}
	if filter.HeightMax != nil {
		conditions = append(conditions, fmt.Sprintf("height <= $%d", argIndex))
		args = append(args, *filter.HeightMax)
		argIndex++
	}
	if filter.Query != nil && *filter.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR bio ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+*filter.Query+"%")
		argIndex++
	}
	where := "WHERE " + strings.Join(conditions, " AND ")
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM model_profiles %s", where)
	var total int
	if err := r.db.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, err
	}
	offset := (pagination.Page - 1) * pagination.Limit
	q := fmt.Sprintf(`SELECT id,user_id,name,bio,description,age,height,weight,gender,clothing_size,shoe_size,experience,
	hourly_rate,city,country,languages,categories,skills,barter_accepted,accept_remote_work,travel_cities,visibility,
	profile_views,rating,total_reviews,is_public,created_at,updated_at FROM model_profiles %s ORDER BY rating DESC, created_at DESC LIMIT $%d OFFSET $%d`, where, argIndex, argIndex+1)
	args = append(args, pagination.Limit, offset)
	var profiles []*ModelProfile
	if err := r.db.SelectContext(ctx, &profiles, q, args...); err != nil {
		return nil, 0, err
	}
	return profiles, total, nil
}
func (r *modelRepository) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE model_profiles SET profile_views = profile_views + 1 WHERE id=$1`, id)
	return err
}
func (r *modelRepository) ListPromoted(ctx context.Context, city *string, limit int) ([]*ModelProfile, error) {
	q := `SELECT DISTINCT ON (p.id)
		p.id,p.user_id,p.name,p.bio,p.description,p.age,p.height,p.weight,p.gender,p.clothing_size,p.shoe_size,p.experience,
		p.hourly_rate,p.city,p.country,COALESCE(p.languages,'[]'::jsonb) as languages,COALESCE(p.categories,'[]'::jsonb) as categories,
		COALESCE(p.skills,'[]'::jsonb) as skills,COALESCE(p.barter_accepted,false) as barter_accepted,COALESCE(p.accept_remote_work,false) as accept_remote_work,
		COALESCE(p.travel_cities,'[]'::jsonb) as travel_cities,COALESCE(p.visibility,'public') as visibility,p.profile_views,p.rating,p.total_reviews,COALESCE(p.is_public,true) as is_public,
		p.created_at,p.updated_at
	FROM model_profiles p
	INNER JOIN profile_promotions pr ON pr.profile_id = p.id
	WHERE p.is_public = true AND pr.status='active' AND pr.starts_at <= NOW() AND pr.ends_at >= NOW()`
	var args []interface{}
	argNum := 1
	if city != nil && *city != "" {
		q += fmt.Sprintf(" AND p.city = $%d", argNum)
		args = append(args, *city)
		argNum++
	}
	q += ` ORDER BY p.id, COALESCE(pr.daily_budget,0) DESC, pr.created_at DESC`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, limit)
	} else {
		q += " LIMIT 20"
	}
	var profiles []*ModelProfile
	if err := r.db.SelectContext(ctx, &profiles, q, args...); err != nil {
		return nil, err
	}
	return profiles, nil
}
func (r *modelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM model_profiles WHERE id=$1`, id)
	return err
}

// employer

type employerRepository struct{ db *sqlx.DB }

func NewEmployerRepository(db *sqlx.DB) EmployerRepository { return &employerRepository{db: db} }
func (r *employerRepository) Create(ctx context.Context, p *EmployerProfile) error {
	q := `INSERT INTO employer_profiles (id,user_id,company_name,company_type,description,website,contact_person,contact_phone,city,country,rating,total_reviews,castings_posted,is_verified,verified_at,created_at,updated_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`
	_, err := r.db.ExecContext(ctx, q, p.ID, p.UserID, p.CompanyName, p.CompanyType, p.Description, p.Website, p.ContactPerson, p.ContactPhone, p.City, p.Country, p.Rating, p.TotalReviews, p.CastingsPosted, p.IsVerified, p.VerifiedAt, p.CreatedAt, p.UpdatedAt)
	return err
}
func (r *employerRepository) GetByID(ctx context.Context, id uuid.UUID) (*EmployerProfile, error) {
	q := `SELECT id,user_id,company_name,company_type,description,website,contact_person,contact_phone,city,country,rating,total_reviews,castings_posted,is_verified,verified_at,created_at,updated_at FROM employer_profiles WHERE id=$1`
	var p EmployerProfile
	err := r.db.GetContext(ctx, &p, q, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func (r *employerRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*EmployerProfile, error) {
	q := `SELECT id,user_id,company_name,company_type,description,website,contact_person,contact_phone,city,country,rating,total_reviews,castings_posted,is_verified,verified_at,created_at,updated_at FROM employer_profiles WHERE user_id=$1`
	var p EmployerProfile
	err := r.db.GetContext(ctx, &p, q, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func (r *employerRepository) Update(ctx context.Context, p *EmployerProfile) error {
	q := `UPDATE employer_profiles SET company_name=$2,company_type=$3,description=$4,website=$5,contact_person=$6,contact_phone=$7,city=$8,country=$9,updated_at=NOW() WHERE id=$1`
	_, err := r.db.ExecContext(ctx, q, p.ID, p.CompanyName, p.CompanyType, p.Description, p.Website, p.ContactPerson, p.ContactPhone, p.City, p.Country)
	return err
}
func (r *employerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM employer_profiles WHERE id=$1`, id)
	return err
}

// admin

type adminRepository struct{ db *sqlx.DB }

func NewAdminRepository(db *sqlx.DB) AdminRepository { return &adminRepository{db: db} }
func (r *adminRepository) Create(ctx context.Context, p *AdminProfile) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO admin_profiles (id,user_id,name,role,avatar_url,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, p.ID, p.UserID, p.Name, p.Role, p.AvatarURL, p.CreatedAt, p.UpdatedAt)
	return err
}
func (r *adminRepository) GetByID(ctx context.Context, id uuid.UUID) (*AdminProfile, error) {
	var p AdminProfile
	err := r.db.GetContext(ctx, &p, `SELECT id,user_id,name,role,avatar_url,created_at,updated_at FROM admin_profiles WHERE id=$1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func (r *adminRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*AdminProfile, error) {
	var p AdminProfile
	err := r.db.GetContext(ctx, &p, `SELECT id,user_id,name,role,avatar_url,created_at,updated_at FROM admin_profiles WHERE user_id=$1`, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func (r *adminRepository) Update(ctx context.Context, p *AdminProfile) error {
	_, err := r.db.ExecContext(ctx, `UPDATE admin_profiles SET name=$2,role=$3,avatar_url=$4,updated_at=NOW() WHERE id=$1`, p.ID, p.Name, p.Role, p.AvatarURL)
	return err
}
