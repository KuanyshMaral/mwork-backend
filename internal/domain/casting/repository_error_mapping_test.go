package casting

import (
	"errors"
	"testing"

	"github.com/lib/pq"
)

func TestMapCreateDBError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantIsErr  error
		wantNotNil bool
	}{
		{name: "pay range check", err: &pq.Error{Code: "23514", Constraint: "valid_pay_range"}, wantIsErr: ErrInvalidPayRange, wantNotNil: true},
		{name: "date range check", err: &pq.Error{Code: "23514", Constraint: "valid_date_range"}, wantIsErr: ErrInvalidDateRange, wantNotNil: true},
		{name: "fk violation", err: &pq.Error{Code: "23503", Constraint: "castings_creator_id_fkey"}, wantIsErr: ErrInvalidCreatorReference, wantNotNil: true},
		{name: "unique violation", err: &pq.Error{Code: "23505", Constraint: "castings_title_key"}, wantIsErr: ErrDuplicateCasting, wantNotNil: true},
		{name: "unknown check", err: &pq.Error{Code: "23514", Constraint: "other_check"}, wantIsErr: ErrCastingConstraint, wantNotNil: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mapped := mapCreateDBError(tc.err)
			if tc.wantNotNil && mapped == nil {
				t.Fatalf("expected mapped error")
			}
			if !errors.Is(mapped, tc.wantIsErr) {
				t.Fatalf("expected errors.Is(%v), got %v", tc.wantIsErr, mapped)
			}
		})
	}
}
