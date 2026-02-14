package robokassa

import (
	"fmt"
	"math/big"
)

func ParseAmount(raw string) (*big.Rat, error) {
	amount, ok := new(big.Rat).SetString(raw)
	if !ok {
		return nil, fmt.Errorf("invalid amount %q", raw)
	}
	return amount, nil
}

func AmountsEqual(expected, actual *big.Rat) bool {
	return expected.Cmp(actual) == 0
}
