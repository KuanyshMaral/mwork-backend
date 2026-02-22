package robokassa

import "testing"

func TestSign_MD5(t *testing.T) {
	sig, err := Sign("abc", HashMD5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig != "900150983cd24fb0d6963f7d28e17f72" {
		t.Fatalf("unexpected hash: %s", sig)
	}
}

func TestNormalizeHashAlgorithm_MD5(t *testing.T) {
	algo, err := NormalizeHashAlgorithm("md5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if algo != HashMD5 {
		t.Fatalf("unexpected algo: %s", algo)
	}
}
