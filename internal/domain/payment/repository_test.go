package payment

import "testing"

func TestIsUndefinedColumnErr(t *testing.T) {
	err := "pq: column payments.robokassa_inv_id does not exist"
	if !isUndefinedColumnErr(assertErr(err), "robokassa_inv_id") {
		t.Fatal("expected undefined column match")
	}
	if isUndefinedColumnErr(assertErr(err), "inv_id") {
		t.Fatal("did not expect different column to match")
	}
}

func TestIsUndefinedPaymentsColumnErr(t *testing.T) {
	err := "pq: column payments.raw_init_payload does not exist"
	if !isUndefinedPaymentsColumnErr(assertErr(err)) {
		t.Fatal("expected payments undefined column error to match")
	}

	nonPaymentsErr := "pq: column users.status does not exist"
	if isUndefinedPaymentsColumnErr(assertErr(nonPaymentsErr)) {
		t.Fatal("did not expect non-payments column error to match")
	}
}

func TestGenerateFallbackRobokassaInvID(t *testing.T) {
	if id := generateFallbackRobokassaInvID(); id <= 0 {
		t.Fatalf("expected positive fallback id, got %d", id)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func assertErr(v string) error { return errString(v) }
