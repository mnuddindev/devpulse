package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/gofiber/fiber/v2"
)

type Map map[string]string

func GenerateOTP() (int64, error) {
	max := big.NewInt(99999999) // 8 digits max
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, fmt.Errorf("failed to generate OTP: %w", err)
	}
	otp := n.Int64()
	// Ensure 8 digits by padding if needed
	if otp < 10000000 {
		otp += 10000000 // Min 10000000
	}
	return otp, nil
}

// StrictBodyParser parses the request body strictly and returns an error if the body contains unknown fields.
func StrictBodyParser(c *fiber.Ctx, out interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields() // Reject unknown fields
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}

// Contains checks if a string exists in a slice of strings.
func Contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
