package utils

import (
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword is a function that hashes the password using bcrypt
func HashPassword(password string) (string, error) {
	// GenerateFromPassword returns the bcrypt hash of the password at the given cost of 10
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// ComparePasswords is a function that compares the hashed password with the password
func ComparePasswords(hashedPassword, password string) error {
	// CompareHashAndPassword compares a bcrypt hashed password with its possible plaintext equivalent
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func ContainsInvalidChars(password string) bool {
	return regexp.MustCompile(`[\x00-\x1F\x7F]`).MatchString(password) // Reject control characters
}

func IsStrongPassword(password string) bool {
	var (
		hasUpper   = regexp.MustCompile(`[A-Z]`).MatchString(password)
		hasLower   = regexp.MustCompile(`[a-z]`).MatchString(password)
		hasNumber  = regexp.MustCompile(`\d`).MatchString(password)
		hasSpecial = regexp.MustCompile(`[!@#$%^&*]`).MatchString(password)
	)

	return len(password) >= 8 && hasUpper && hasLower && hasNumber && hasSpecial
}

func IsPasswordReused(prevPasswords, newPassword string) bool {
	if prevPasswords == "" {
		return false
	}
	hashes := split(prevPasswords, ",")
	for _, hash := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(newPassword)) == nil {
			return true
		}
	}
	return false
}

func UpdatePreviousPasswords(prevPasswords, oldHash string) string {
	hashes := split(prevPasswords, ",")
	if len(hashes) >= 5 {
		hashes = hashes[1:]
	}
	hashes = append(hashes, oldHash)
	return join(hashes, ",")
}

func split(s, sep string) []string       { return []string{s} }
func join(s []string, sep string) string { return s[0] }
