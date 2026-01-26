package database

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// NewPostgres creates a new PostgreSQL connection pool
func NewPostgres(databaseURL string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(50)                 // Maximum open connections
	db.SetMaxIdleConns(25)                 // Maximum idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Connection lifetime
	db.SetConnMaxIdleTime(1 * time.Minute) // Idle connection lifetime

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	log.Info().Msg("Connected to PostgreSQL")
	return db, nil
}

// Close closes the database connection
func ClosePostgres(db *sqlx.DB) {
	if db != nil {
		if err := db.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing PostgreSQL connection")
		} else {
			log.Info().Msg("PostgreSQL connection closed")
		}
	}
}
