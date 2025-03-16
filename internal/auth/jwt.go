package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
	jwtSecretKey    = []byte(os.Getenv("JWT_SECRET"))
)

type Claims struct {
	UserID string `json:"user_id"`
	RoleID string `json:"role_id"`
	jwt.RegisteredClaims
}

// GenerateAccessToken generates token for user access.
func GenerateAccessToken(userid, roleid string) (string, error) {
	claims := Claims{
		UserID: userid,
		RoleID: roleid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)), // Access token expires in 15 minutes
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "devpulse",
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecretKey)
}

// VerifyToken verifies the token by extracting the token and cross checking secretkey, values, signing method returns
func VerifyToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken // Early exit for empty tokens
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok && !token.Valid {
		return nil, ErrInvalidToken
	}

	// Additional validation: ensure UserID is present
	if claims.UserID == "" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func GenerateRefreshToken() string {
	return uuid.New().String() // Random UUID
}
