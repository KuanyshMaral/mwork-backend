package upload

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
)

func TestIsUploadUserReferenceError(t *testing.T) {
	t.Run("matches uploads user fk violation", func(t *testing.T) {
		err := &pq.Error{Code: "23503", Constraint: "uploads_user_id_fkey"}
		if !isUploadUserReferenceError(err) {
			t.Fatal("expected true for uploads_user_id_fkey foreign key violation")
		}
	})

	t.Run("different foreign key constraint", func(t *testing.T) {
		err := &pq.Error{Code: "23503", Constraint: "other_fkey"}
		if isUploadUserReferenceError(err) {
			t.Fatal("expected false for other constraint")
		}
	})

	t.Run("wrapped matching error", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &pq.Error{Code: "23503", Constraint: "uploads_user_id_fkey"})
		if !isUploadUserReferenceError(err) {
			t.Fatal("expected true for wrapped pq.Error")
		}
	})

	t.Run("non-pq error", func(t *testing.T) {
		err := errors.New("boom")
		if isUploadUserReferenceError(err) {
			t.Fatal("expected false for non-pq error")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		if isUploadUserReferenceError(nil) {
			t.Fatal("expected false for nil error")
		}
	})
}
