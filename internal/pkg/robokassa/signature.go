package robokassa

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type HashAlgorithm string

const (
	HashMD5    HashAlgorithm = "MD5"
	HashSHA256 HashAlgorithm = "SHA256"
)

func NormalizeHashAlgorithm(raw string) (HashAlgorithm, error) {
	algo := HashAlgorithm(strings.ToUpper(strings.TrimSpace(raw)))
	switch algo {
	case HashMD5:
		return algo, nil
	case HashSHA256:
		return algo, nil
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", raw)
	}
}

func BuildStartSignatureBase(merchantLogin, outSum, invID, password1 string, receipt *string, shp map[string]string) string {
	parts := []string{merchantLogin, outSum, invID}
	if receipt != nil {
		parts = append(parts, url.QueryEscape(*receipt))
	}
	parts = append(parts, password1)
	parts = append(parts, sortedShpPairs(shp)...)
	return strings.Join(parts, ":")
}

func BuildResultSignatureBase(outSum, invID, password2 string, shp map[string]string) string {
	parts := []string{outSum, invID, password2}
	parts = append(parts, sortedShpPairs(shp)...)
	return strings.Join(parts, ":")
}

func BuildSuccessSignatureBase(outSum, invID, password1 string, shp map[string]string) string {
	parts := []string{outSum, invID, password1}
	parts = append(parts, sortedShpPairs(shp)...)
	return strings.Join(parts, ":")
}

func Sign(base string, algo HashAlgorithm) (string, error) {
	switch algo {
	case HashMD5:
		h := md5.Sum([]byte(base))
		return hex.EncodeToString(h[:]), nil
	case HashSHA256:
		h := sha256.Sum256([]byte(base))
		return hex.EncodeToString(h[:]), nil
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algo)
	}
}

func VerifySignature(expectedHex, receivedHex string) bool {
	expected := strings.ToLower(strings.TrimSpace(expectedHex))
	received := strings.ToLower(strings.TrimSpace(receivedHex))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(received)) == 1
}

func sortedShpPairs(shp map[string]string) []string {
	if len(shp) == 0 {
		return nil
	}

	keys := make([]string, 0, len(shp))
	for k := range shp {
		if strings.HasPrefix(strings.ToLower(k), "shp_") {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, url.QueryEscape(shp[key])))
	}
	return pairs
}
