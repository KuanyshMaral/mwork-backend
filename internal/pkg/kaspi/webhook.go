package kaspi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// VerifySignature validates HMAC-SHA256 signature from Kaspi webhook
// Returns true if signature matches expected value
func VerifySignature(payload []byte, signature string, secretKey string) bool {
	if secretKey == "" || signature == "" {
		return false
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write(payload)
	expected := h.Sum(nil)

	given, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	return hmac.Equal(given, expected)
}

// GenerateSignature creates HMAC-SHA256 signature for testing
func GenerateSignature(payload []byte, secretKey string) string {
	if secretKey == "" {
		return ""
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
