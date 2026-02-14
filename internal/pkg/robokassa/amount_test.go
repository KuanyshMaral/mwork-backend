package robokassa

import "testing"

func TestAmountsEqual_DifferentScale(t *testing.T) {
	a, _ := ParseAmount("100.10")
	b, _ := ParseAmount("100.100000")
	if !AmountsEqual(a, b) {
		t.Fatal("amounts should be numerically equal")
	}
}
