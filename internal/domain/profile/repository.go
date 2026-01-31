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

// ---- MODEL REPOSITORY ----

type modelRepository struct {
	db *sqlx.DB
}

// NewModelRepository creates new model profile repository
func NewModelRepository(db *sqlx.DB) ModelRepository {
	return &modelRepository{db: db}
}

func (r *modelRepository) Create(ctx context.Context, profile *ModelProfile) error {
	query := `
		INSERT INTO profiles (
			id, user_id, type, first_name, bio, description, age, height_cm, weight_kg, gender,
			clothing_size, shoe_size, experience_years, hourly_rate, city, country,
			languages, categories, skills, barter_accepted, accept_remote_work,
			travel_cities, visibility,
			view_count, rating, total_reviews, is_public
		) VALUES (
			$1, $2, 'model', $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20,
			$21, $22,
			$23, $24, $25, $26
		)
	`

	// Note: Mapping Name -> first_name, we assume Name holds fullname for now or just first name
	// In the split, we used 'Name' but DB has first_name, last_name. We'll map Name -> first_name

	_, err := r.db.ExecContext(ctx, query,
		profile.ID, profile.UserID, profile.Name, profile.Bio, profile.Description,
		profile.Age, profile.Height, profile.Weight, profile.Gender,
		profile.ClothingSize, profile.ShoeSize, profile.Experience, profile.HourlyRate,
		profile.City, profile.Country,
		profile.Languages, profile.Categories, profile.Skills,
		profile.BarterAccepted, profile.AcceptRemoteWork,
		profile.TravelCities, profile.Visibility,
		profile.ProfileViews, profile.Rating, profile.TotalReviews, profile.IsPublic,
	)

	return err
}

func (r *modelRepository) GetByID(ctx context.Context, id uuid.UUID) (*ModelProfile, error) {
	// Need to map DB columns (first_name, height_cm, etc) back to Struct fields (Name, Height, etc)
	query := `
		SELECT 
			id, user_id, first_name as name, bio, description, age, height_cm as height, weight_kg as weight, gender,
			clothing_size, shoe_size, experience_years as experience, hourly_rate, city, country,
			languages, categories, skills, barter_accepted, accept_remote_work,
			travel_cities, visibility,
			view_count as profile_views, rating, total_reviews, is_public, created_at, updated_at

		FROM profiles 
		WHERE id = $1 AND type = 'model'
	`

	var profile ModelProfile
	err := r.db.GetContext(ctx, &profile, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &profile, nil
}

func (r *modelRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*ModelProfile, error) {
	query := `
		SELECT 
			id, user_id, first_name as name, bio, description, age, height_cm as height, weight_kg as weight, gender,
			clothing_size, shoe_size, experience_years as experience, hourly_rate, city, country,
			languages, categories, skills, barter_accepted, accept_remote_work,
			travel_cities, visibility,
			view_count as profile_views, rating, total_reviews, is_public, created_at, updated_at
		FROM profiles 
		WHERE user_id = $1 AND type = 'model'
	`

	var profile ModelProfile
	err := r.db.GetContext(ctx, &profile, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &profile, nil
}

func (r *modelRepository) Update(ctx context.Context, profile *ModelProfile) error {
	query := `
		UPDATE profiles SET
			first_name = $2, bio = $3, description = $4, age = $5, height_cm = $6, weight_kg = $7, gender = $8,
			clothing_size = $9, shoe_size = $10, experience_years = $11, hourly_rate = $12,
			city = $13, country = $14, languages = $15, categories = $16, skills = $17,
			barter_accepted = $18, accept_remote_work = $19, is_public = $20, travel_cities = $21, visibility = $22,
			updated_at = NOW()
		WHERE id = $1 AND type = 'model'
	`

	_, err := r.db.ExecContext(ctx, query,
		profile.ID,
		profile.Name, profile.Bio, profile.Description, profile.Age, profile.Height, profile.Weight, profile.Gender,
		profile.ClothingSize, profile.ShoeSize, profile.Experience, profile.HourlyRate,
		profile.City, profile.Country, profile.Languages, profile.Categories, profile.Skills,
		profile.BarterAccepted, profile.AcceptRemoteWork, profile.IsPublic, profile.TravelCities, profile.Visibility,
	)

	return err
}

func (r *modelRepository) List(ctx context.Context, filter *Filter, pagination *Pagination) ([]*ModelProfile, int, error) {
	conditions := []string{"type = 'model'", "is_public = true"}
	args := []interface{}{}
	argIndex := 1

	if filter.City != nil && *filter.City != "" {
		conditions = append(conditions, fmt.Sprintf("city ILIKE $%d", argIndex))
		args = append(args, "%"+*filter.City+"%")
		argIndex++
	}

	if filter.Gender != nil && *filter.Gender != "" {
		conditions = append(conditions, fmt.Sprintf("gender = $%d", argIndex))
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
		conditions = append(conditions, fmt.Sprintf("height_cm >= $%d", argIndex))
		args = append(args, *filter.HeightMin)
		argIndex++
	}

	if filter.HeightMax != nil {
		conditions = append(conditions, fmt.Sprintf("height_cm <= $%d", argIndex))
		args = append(args, *filter.HeightMax)
		argIndex++
	}

	if filter.Query != nil && *filter.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(first_name ILIKE $%d OR bio ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+*filter.Query+"%")
		argIndex++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM profiles %s", where)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// Get profiles with pagination
	offset := (pagination.Page - 1) * pagination.Limit
	query := fmt.Sprintf(`
		SELECT 
			id, user_id, first_name as name, bio, description, age, height_cm as height, weight_kg as weight, gender,
			clothing_size, shoe_size, experience_years as experience, hourly_rate, city, country,
			languages, categories, skills, barter_accepted, accept_remote_work,
			travel_cities, visibility,
			view_count as profile_views, rating, total_reviews, is_public, created_at, updated_at
		FROM profiles %s 
		ORDER BY rating DESC, created_at DESC 
		LIMIT $%d OFFSET $%d
	`, where, argIndex, argIndex+1)
	args = append(args, pagination.Limit, offset)

	var profiles []*ModelProfile
	if err := r.db.SelectContext(ctx, &profiles, query, args...); err != nil {
		return nil, 0, err
	}

	return profiles, total, nil
}

func (r *modelRepository) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE profiles SET view_count = view_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *modelRepository) ListPromoted(ctx context.Context, city *string, limit int) ([]*ModelProfile, error) {
	// Join with profile_promotions table to get promoted profiles
	// Priority: active promotions first, sorted by daily_budget DESC, then created_at DESC
	query := `
		SELECT DISTINCT ON (p.id)
			p.id, p.user_id, p.first_name as name, p.bio, p.description,
			p.age, p.height_cm as height, p.weight_kg as weight, p.gender,
			p.clothing_size, p.shoe_size, p.experience_years as experience,
			p.hourly_rate, p.city, p.country,
			COALESCE(p.languages, '{}') as languages,
			COALESCE(p.categories, '{}') as categories,
			COALESCE(p.skills, '{}') as skills,
			COALESCE(p.barter_accepted, false) as barter_accepted,
			COALESCE(p.accept_remote_work, false) as accept_remote_work,
			COALESCE(p.travel_cities, '{}') as travel_cities,
			COALESCE(p.visibility, 'public') as visibility,
			p.view_count as profile_views, p.rating, p.total_reviews,
			COALESCE(p.is_public, true) as is_public,
			p.created_at, p.updated_at
		FROM profiles p
		INNER JOIN profile_promotions pr ON pr.profile_id = p.id
		WHERE p.type = 'model'
			AND p.is_public = true
			AND pr.status = 'active'
			AND pr.starts_at <= NOW()
			AND pr.ends_at >= NOW()
	`

	var args []interface{}
	argNum := 1

	if city != nil && *city != "" {
		query += fmt.Sprintf(" AND p.city = $%d", argNum)
		args = append(args, *city)
		argNum++
	}

	query += `
		ORDER BY p.id,
			COALESCE(pr.daily_budget, 0) DESC,
			pr.created_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, limit)
	} else {
		query += " LIMIT 20" // Default limit
	}

	var profiles []*ModelProfile
	if err := r.db.SelectContext(ctx, &profiles, query, args...); err != nil {
		return nil, err
	}

	return profiles, nil
}

func (r *modelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Delete associated portfolio items first
	// Note: portfolio_items likely uses 'profile_id', check schema later. Assuming model_id based on old code.
	// Actually schema for photos is 'photos' table usually linked to 'profiles'.
	// Checking 000005_create_photos: 'profile_id REFERENCES profiles(id)'

	// We rely on CASCADE usually, but explicit delete is safer
	_, err := r.db.ExecContext(ctx, `DELETE FROM profiles WHERE id = $1 AND type='model'`, id)
	return err
}

// ---- EMPLOYER REPOSITORY ----

type employerRepository struct {
	db *sqlx.DB
}

// NewEmployerRepository creates new employer profile repository
func NewEmployerRepository(db *sqlx.DB) EmployerRepository {
	return &employerRepository{db: db}
}

func (r *employerRepository) Create(ctx context.Context, profile *EmployerProfile) error {
	query := `
		INSERT INTO profiles (
			id, user_id, type, company_name, company_type, description, website,
			contact_person, contact_phone, city, country,
			rating, total_reviews, is_public, is_verified
		) VALUES (
			$1, $2, 'employer', $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, true, $13
		)
	`
	// Note: EmployerProfile has CastingsPosted? 'profiles' table doesn't seem to have it.
	// 000002_create_profiles.up.sql doesn't show castings_posted.
	// We will omit CastingsPosted from INSERT.

	_, err := r.db.ExecContext(ctx, query,
		profile.ID, profile.UserID, profile.CompanyName, profile.CompanyType,
		profile.Description, profile.Website,
		profile.ContactPerson, profile.ContactPhone, profile.City, profile.Country,
		profile.Rating, profile.TotalReviews, profile.IsVerified,
	)

	return err
}

func (r *employerRepository) GetByID(ctx context.Context, id uuid.UUID) (*EmployerProfile, error) {
	query := `
		SELECT 
			id, user_id, company_name, company_type, description, website,
			contact_person, contact_phone, city, country,
			rating, total_reviews, is_verified, created_at, updated_at
		FROM profiles 
		WHERE id = $1 AND type = 'employer'
	`
	// Missing: castings_posted. We can calculate it or leave 0.

	var profile EmployerProfile
	err := r.db.GetContext(ctx, &profile, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &profile, nil
}

func (r *employerRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*EmployerProfile, error) {
	query := `
		SELECT 
			id, user_id, company_name, company_type, description, website,
			contact_person, contact_phone, city, country,
			rating, total_reviews, is_verified, created_at, updated_at
		FROM profiles 
		WHERE user_id = $1 AND type = 'employer'
	`

	var profile EmployerProfile
	err := r.db.GetContext(ctx, &profile, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &profile, nil
}

func (r *employerRepository) Update(ctx context.Context, profile *EmployerProfile) error {
	query := `
		UPDATE profiles SET
			company_name = $2, company_type = $3, description = $4, website = $5,
			contact_person = $6, contact_phone = $7, city = $8, country = $9,
			updated_at = NOW()
		WHERE id = $1 AND type = 'employer'
	`

	_, err := r.db.ExecContext(ctx, query,
		profile.ID,
		profile.CompanyName, profile.CompanyType, profile.Description, profile.Website,
		profile.ContactPerson, profile.ContactPhone, profile.City, profile.Country,
	)

	return err
}

func (r *employerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM profiles WHERE id = $1 AND type='employer'`, id)
	return err
}
