package robokassa

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestCreatePayment_ShpPrefixedKeyNotDoublePrefixed(t *testing.T) {
	client := NewClient(Config{
		MerchantLogin: "merchant",
		Password1:     "p1",
		HashAlgo:      HashSHA256,
	})

	resp, err := client.CreatePayment(context.Background(), CreatePaymentRequest{
		Amount: 100,
		InvID:  1,
		Shp: map[string]string{
			"Shp_order": "42",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := url.Parse(resp.PaymentURL)
	if err != nil {
		t.Fatalf("failed to parse url: %v", err)
	}
	q := u.Query()
	if q.Get("Shp_order") != "42" {
		t.Fatalf("expected Shp_order param, got query: %s", u.RawQuery)
	}
	if strings.Contains(u.RawQuery, "Shp_Shp_order") {
		t.Fatalf("unexpected double-prefixed shp parameter: %s", u.RawQuery)
	}
}
