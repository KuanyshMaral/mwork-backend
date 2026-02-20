package payment

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
)

type RobokassaService struct {
	MerchantLogin string
	Password1     string
	Password2     string
	BaseURL       string
}

func (s RobokassaService) GeneratePaymentLink(outSum string, invID string) string {
	sig := s.sign(strings.Join([]string{s.MerchantLogin, outSum, invID, s.Password1}, ":"))
	params := url.Values{}
	params.Set("MerchantLogin", s.MerchantLogin)
	params.Set("OutSum", outSum)
	params.Set("InvId", invID)
	params.Set("SignatureValue", sig)
	return strings.TrimRight(s.BaseURL, "?") + "?" + params.Encode()
}

func (s RobokassaService) ValidateResultSignature(outSum, invID, signature string) bool {
	expected := s.sign(strings.Join([]string{outSum, invID, s.Password2}, ":"))
	return strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(signature))
}

func (s RobokassaService) sign(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
