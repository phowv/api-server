package random

import (
	"crypto/rand"
	"math/big"
)

func CryptoRandInt64(left, right int64) (int64, error) {
	n := right - left + 1
	bi, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		return 0, err
	}
	return bi.Int64() + left, nil
}

func CryptoRandInt(left, right int) (int, error) {
	bi, err := CryptoRandInt64(int64(left), int64(right))
	return int(bi), err
}
