package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"math/big"

	"github.com/gofiber/fiber/v2"
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
