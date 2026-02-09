package robokassa

import (
	"fmt"
	"strconv"
	"strings"
)

// WebhookPayload represents RoboKassa webhook (ResultURL) data
// RoboKassa sends data as form parameters, not JSON
type WebhookPayload struct {
	OutSum         string            // Payment amount
	InvId          int64             // Invoice ID
	SignatureValue string            // Signature to verify (CRC)
	Shp            map[string]string // Custom parameters
}

// VerifyResultSignature validates webhook signature from RoboKassa ResultURL
// Signature format: MD5(OutSum:InvId:Password2[:Shp_params])
func VerifyResultSignature(outSum string, invID int64, signature string, password2 string, shpParams map[string]string) bool {
	if password2 == "" || signature == "" {
		return false
	}

	// Build signature string
	signatureStr := fmt.Sprintf("%s:%d:%s", outSum, invID, password2)

	// Add custom parameters in alphabetical order
	if len(shpParams) > 0 {
		var keys []string
		for k := range shpParams {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			signatureStr += fmt.Sprintf(":%s=%s", k, shpParams[k])
		}
	}

	expected := generateMD5(signatureStr)
	given := strings.ToUpper(signature)

	return expected == given
}

// VerifySuccessSignature validates signature for SuccessURL
// Same as ResultURL but different purpose (user redirect vs server notification)
func VerifySuccessSignature(outSum string, invID int64, signature string, password1 string, shpParams map[string]string) bool {
	if password1 == "" || signature == "" {
		return false
	}

	signatureStr := fmt.Sprintf("%s:%d:%s", outSum, invID, password1)

	if len(shpParams) > 0 {
		var keys []string
		for k := range shpParams {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			signatureStr += fmt.Sprintf(":%s=%s", k, shpParams[k])
		}
	}

	expected := generateMD5(signatureStr)
	given := strings.ToUpper(signature)

	return expected == given
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

	// Extract custom parameters (Shp_*)
	shp := make(map[string]string)
	for key, values := range formValues {
		if strings.HasPrefix(strings.ToLower(key), "shp_") {
			paramName := strings.TrimPrefix(strings.ToLower(key), "shp_")
			if len(values) > 0 {
				shp[paramName] = values[0]
			}
		}
	}

	return &WebhookPayload{
		OutSum:         outSumStr,
		InvId:          invID,
		SignatureValue: signature,
		Shp:            shp,
	}, nil
}

// Helper functions are in utils.go
