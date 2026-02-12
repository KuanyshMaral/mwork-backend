package robokassa

import "testing"

func TestBuildStartSignatureBase_SortedShpAndEncoded(t *testing.T) {
	receipt := `{"items":[{"name":"Plan Pro"}]}`
	base := BuildStartSignatureBase(
		"merchant",
		"100.50",
		"42",
		"pass1",
		&receipt,
		map[string]string{
			"Shp_user": "user+1",
			"Shp_pay":  "p/42",
		},
	)

	expected := "merchant:100.50:42:%7B%22items%22%3A%5B%7B%22name%22%3A%22Plan+Pro%22%7D%5D%7D:pass1:Shp_pay=p%2F42:Shp_user=user%2B1"
	if base != expected {
		t.Fatalf("unexpected base string:\nwant %s\ngot  %s", expected, base)
	}
}

func TestVerifySignature_CaseInsensitive(t *testing.T) {
	if !VerifySignature("aBcD", "ABcd") {
		t.Fatal("expected case-insensitive constant-time comparison")
	}
}

func TestSign_SHA256(t *testing.T) {
	sig, err := Sign("abc", HashSHA256)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatalf("unexpected hash: %s", sig)
	}
}
