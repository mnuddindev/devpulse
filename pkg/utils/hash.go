package utils

import "golang.org/x/crypto/bcrypt"

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
