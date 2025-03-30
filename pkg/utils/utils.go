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

const (
	numbers = "0123456789"
	letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
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

func GenerateOTP(length int) (string, error) {
	if length < 6 {
		return "", fmt.Errorf("length must be at least 6")
	}

	// Randomly decide the number of letters (max 5)
	maxLetters := 5
	letterCountLimit := big.NewInt(int64(min(length, maxLetters)))
	letterCountBig, err := rand.Int(rand.Reader, letterCountLimit)
	if err != nil {
		return "", err
	}
	letterCount := int(letterCountBig.Int64())

	// Remaining characters will be numbers
	numCount := length - letterCount

	numPart := make([]byte, numCount)
	letterPart := make([]byte, letterCount)

	for i := 0; i < numCount; i++ {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(numbers))))
		if err != nil {
			return "", err
		}
		numPart[i] = numbers[index.Int64()]
	}

	for i := 0; i < letterCount; i++ {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		letterPart[i] = letters[index.Int64()]
	}

	// Combine both parts
	mixed := append(numPart, letterPart...)

	// Securely shuffle the mixed slice
	ShuffleSecure(mixed)

	return string(mixed), nil
}

// shuffleSecure securely shuffles a byte slice using crypto/rand
func ShuffleSecure(data []byte) {
	for i := len(data) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			continue // If we fail to generate a random index, skip to the next iteration
		}
		data[i], data[j.Int64()] = data[j.Int64()], data[i]
	}
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
