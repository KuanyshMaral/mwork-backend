package robokassa

import (
	"fmt"
	"strconv"
	"strings"
)

// defaultAlgo is the algorithm used when none is specified.
// Robokassa recommends SHA256 for all new integrations.
const defaultAlgo = HashSHA256

// WebhookPayload represents RoboKassa webhook (ResultURL) data
// RoboKassa sends data as form parameters, not JSON
type WebhookPayload struct {
	OutSum         string            // Payment amount
	InvId          int64             // Invoice ID
	SignatureValue string            // Signature to verify (CRC)
	Shp            map[string]string // Custom parameters
}

// VerifyResultSignature validates webhook signature from RoboKassa ResultURL
// Signature format: SHA256(OutSum:InvId:Password2[:Shp_params])
// Uses SHA256 as required by Robokassa for new integrations.
func VerifyResultSignature(outSum string, invID int64, signature string, password2 string, shpParams map[string]string) bool {
	return VerifyResultSignatureWithAlgo(outSum, invID, signature, password2, shpParams, defaultAlgo)
}

// VerifyResultSignatureWithAlgo validates webhook signature with a specific hash algorithm.
// Use this when the algorithm is configurable (e.g., from config.HashAlgo).
func VerifyResultSignatureWithAlgo(outSum string, invID int64, signature string, password2 string, shpParams map[string]string, algo HashAlgorithm) bool {
	if password2 == "" || signature == "" {
		return false
	}

	// Build signature base: OutSum:InvId:Password2[:Shp_params]
	base := BuildResultSignatureBase(outSum, strconv.FormatInt(invID, 10), password2, shpParams)

	expected, err := Sign(base, algo)
	if err != nil {
		return false
	}

	// Case-insensitive comparison
	return VerifySignature(expected, signature)
}

// VerifySuccessSignature validates signature for SuccessURL
// Same as ResultURL but different purpose (user redirect vs server notification).
// Uses SHA256 as required by Robokassa for new integrations.
func VerifySuccessSignature(outSum string, invID int64, signature string, password1 string, shpParams map[string]string) bool {
	return VerifySuccessSignatureWithAlgo(outSum, invID, signature, password1, shpParams, defaultAlgo)
}

// VerifySuccessSignatureWithAlgo validates SuccessURL signature with a specific hash algorithm.
func VerifySuccessSignatureWithAlgo(outSum string, invID int64, signature string, password1 string, shpParams map[string]string, algo HashAlgorithm) bool {
	if password1 == "" || signature == "" {
		return false
	}

	// Build signature base: OutSum:InvId:Password1[:Shp_params]
	base := BuildSuccessSignatureBase(outSum, strconv.FormatInt(invID, 10), password1, shpParams)

	expected, err := Sign(base, algo)
	if err != nil {
		return false
	}

	// Case-insensitive comparison
	return VerifySignature(expected, signature)
}

// ParseWebhookForm parses form-encoded webhook data into structured payload
func ParseWebhookForm(formValues map[string][]string) (*WebhookPayload, error) {
	// Extract required fields
	outSumStr := getFirstValue(formValues, "OutSum")
	invIDStr := getFirstValue(formValues, "InvId")
	signature := getFirstValue(formValues, "SignatureValue")

	if outSumStr == "" {
		return nil, fmt.Errorf("OutSum is required")
	}
	if invIDStr == "" {
		return nil, fmt.Errorf("InvId is required")
	}
	if signature == "" {
		return nil, fmt.Errorf("SignatureValue is required")
	}

	invID, err := strconv.ParseInt(invIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid InvId: %w", err)
	}

	// Extract custom parameters (Shp_*), preserving original key casing
	// because key casing is part of the signature base string.
	shp := make(map[string]string)
	for key, values := range formValues {
		if !strings.HasPrefix(strings.ToLower(key), "shp_") || len(values) == 0 {
			continue
		}
		shp[key] = values[0]
	}

	return &WebhookPayload{
		OutSum:         outSumStr,
		InvId:          invID,
		SignatureValue: signature,
		Shp:            shp,
	}, nil
}

// Helper functions are in utils.go
