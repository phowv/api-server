package random

import (
	"fmt"
	"strconv"
)

func GenerateVerificationCode() (string, error) {
	code, err := CryptoRandInt64(100000, 1000000)

	if err != nil {
		return "", fmt.Errorf("failed to generate random number: %w", err)
	}

	return strconv.FormatInt(code, 10), nil
}
