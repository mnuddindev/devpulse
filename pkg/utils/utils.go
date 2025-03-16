package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/gofiber/fiber/v2"
)

type Map map[string]string

// GenerateRandomToken generates a random token with a length between minLength and maxLength.
func GenerateRandomToken(minLength, maxLength int) (string, error) {
	lengthRange := big.NewInt(int64(maxLength - minLength + 1))
	randomOffset, err := rand.Int(rand.Reader, lengthRange)
	if err != nil {
		return "", fmt.Errorf("failed to generate random length: %v", err)
	}
	length := minLength + int(randomOffset.Int64())

	byteLength := (length * 6) / 8
	if (length*6)%8 != 0 {
		byteLength++
	}

	randomBytes := make([]byte, byteLength)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %v", err)
	}

	token := base64.URLEncoding.EncodeToString(randomBytes)

	if len(token) > length {
		token = token[:length]
	}

	return token, nil
}

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
