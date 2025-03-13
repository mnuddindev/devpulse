package utils

import (
	"crypto/rand"
	"math/big"
)

type Map map[string]string

func GenerateOTP() (int64, error) {
	max := big.NewInt(999999)
	numb, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return numb.Int64(), nil
}
