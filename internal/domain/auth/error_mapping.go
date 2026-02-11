package auth

import (
	"errors"
	"fmt"

	"github.com/lib/pq"
)

const (
	sqlStateUniqueViolation = "23505"
)

// DBErrorDetails contains diagnostics extracted from PostgreSQL errors.
type DBErrorDetails struct {
	SQLState   string
	Constraint string
	Table      string
	Column     string
	Detail     string
	Message    string
}

func extractDBErrorDetails(err error) *DBErrorDetails {
	if err == nil {
		return nil
	}

	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return nil
	}

	return &DBErrorDetails{
		SQLState:   string(pqErr.Code),
		Constraint: pqErr.Constraint,
		Table:      pqErr.Table,
		Column:     pqErr.Column,
		Detail:     pqErr.Detail,
		Message:    pqErr.Message,
	}
}

func isEmailAlreadyExistsError(err error) bool {
	if errors.Is(err, ErrEmailAlreadyExists) {
		return true
	}
	details := extractDBErrorDetails(err)
	if details == nil {
		return false
	}
	if details.SQLState != sqlStateUniqueViolation {
		return false
	}
	if details.Constraint == "users_email_key" {
		return true
	}
	if details.Table == "users" && details.Column == "email" {
		return true
	}
	return false
}

func wrapRegisterError(step string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("register step %s: %w", step, err)
}
