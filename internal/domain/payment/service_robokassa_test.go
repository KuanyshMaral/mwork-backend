package payment

import "testing"

func TestIsTestCallback(t *testing.T) {
	if !isTestCallback(map[string]string{"IsTest": "1"}) {
		t.Fatal("expected true for IsTest=1")
	}
	if isTestCallback(map[string]string{"IsTest": "0"}) {
		t.Fatal("expected false for IsTest=0")
	}
}

func TestNormalizeAmount_Comma(t *testing.T) {
	a, err := normalizeAmount("100,50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.FloatString(2) != "100.50" {
		t.Fatalf("unexpected normalized amount: %s", a.FloatString(2))
	}
}
