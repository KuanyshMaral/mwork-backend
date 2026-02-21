package payment

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/mwork/mwork-api/internal/pkg/robokassa"
)

type RobokassaService struct {
	MerchantLogin string
	Password1     string
	Password2     string
	BaseURL       string
	HashAlgo      robokassa.HashAlgorithm
}

func (s RobokassaService) GeneratePaymentLink(outSum string, invID string) (string, error) {
	algo := s.HashAlgo
	if algo == "" {
		algo = robokassa.HashSHA256
	}
	base := robokassa.BuildStartSignatureBase(s.MerchantLogin, outSum, invID, s.Password1, nil, nil)
	signature, err := robokassa.Sign(base, algo)
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Set("MerchantLogin", s.MerchantLogin)
	params.Set("OutSum", outSum)
	params.Set("InvId", invID)
	params.Set("SignatureValue", signature)
	baseURL := strings.TrimSpace(s.BaseURL)
	if baseURL == "" {
		baseURL = "https://auth.robokassa.kz/Merchant/Index.aspx"
	}
	return strings.TrimRight(baseURL, "?") + "?" + params.Encode(), nil
}

func (s RobokassaService) ValidateResultSignature(outSum, invID, signature string, shp map[string]string) bool {
	algo := s.HashAlgo
	if algo == "" {
		algo = robokassa.HashSHA256
	}
	parsedInvID, err := strconv.ParseInt(strings.TrimSpace(invID), 10, 64)
	if err != nil {
		return false
	}
	return robokassa.VerifyResultSignatureWithAlgo(outSum, parsedInvID, signature, s.Password2, shp, algo)
}
