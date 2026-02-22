package robokassa

import "testing"

func TestParseWebhookForm_PreservesShpKeyCase(t *testing.T) {
	payload, err := ParseWebhookForm(map[string][]string{
		"OutSum":         {"100.00"},
		"InvId":          {"42"},
		"SignatureValue": {"sig"},
		"Shp_orderId":    {"A-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.Shp["Shp_orderId"] != "A-1" {
		t.Fatalf("expected original shp key preserved, got: %#v", payload.Shp)
	}
}
