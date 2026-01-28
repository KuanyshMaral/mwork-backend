package experience

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// Entity представляет запись опыта работы
type Entity struct {
	ID          string `db:"id" json:"id"`
	ProfileID   string `db:"profile_id" json:"profile_id"`
	Title       string `db:"title" json:"title"`
	Company     string `db:"company" json:"company"`
	Role        string `db:"role" json:"role"`
	Year        int    `db:"year" json:"year"`
	Description string `db:"description" json:"description"`
	CreatedAt   string `db:"created_at" json:"created_at"`
}

// Repository интерфейс для работы с опытом работы
type Repository interface {
	Create(ctx context.Context, exp *Entity) error
	ListByProfileID(ctx context.Context, profileID string) ([]*Entity, error)
	Delete(ctx context.Context, id string) error
}

type repositoryImpl struct {
	db *sqlx.DB
}

// NewRepository создает новый репозиторий для работы с опытом работы
func NewRepository(db *sqlx.DB) Repository {
	return &repositoryImpl{db: db}
}

// Create вставляет новый опыт работы
func (r *repositoryImpl) Create(ctx context.Context, exp *Entity) error {
	// Выполняем запрос на вставку
	query := `INSERT INTO work_experiences (profile_id, title, company, role, year, description)
	          VALUES ($1, $2, $3, $4, $5, $6)
	          RETURNING id, created_at`

	return r.db.QueryRowContext(ctx, query, exp.ProfileID, exp.Title, exp.Company, exp.Role, exp.Year, exp.Description).
		Scan(&exp.ID, &exp.CreatedAt)
}

// ListByProfileID возвращает все записи опыта работы для указанного профиля
func (r *repositoryImpl) ListByProfileID(ctx context.Context, profileID string) ([]*Entity, error) {
	// Выполняем запрос на выборку
	query := `SELECT * FROM work_experiences WHERE profile_id = $1 ORDER BY year DESC`

	var experiences []*Entity
	err := r.db.SelectContext(ctx, &experiences, query, profileID)
	if err != nil {
		return nil, err
	}
	return experiences, nil
}

// Delete удаляет запись о работе по ID
func (r *repositoryImpl) Delete(ctx context.Context, id string) error {
	// Выполняем запрос на удаление
	query := `DELETE FROM work_experiences WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
