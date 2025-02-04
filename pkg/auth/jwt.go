package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/config"
	"github.com/sirupsen/logrus"
)

var (
	ErrInvalidToken = errors.New("Invalid token")
	ErrExpiredToken = errors.New("token has expired")
	jwtSecretKey    = []byte(config.ServerConfig.JWTSecret)
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(userID uuid.UUID, email string) string {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)), // Access token expires in 15 minutes
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString(jwtSecretKey)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
			"field": "GenerateAccessToken",
		}).Error("Access Token generation failed!!")
	}
	return t
}

// GenerateRefreshToken generates a new refresh token
func GenerateRefreshToken(userID uuid.UUID) string {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // Refresh token expires in 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString(jwtSecretKey)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
			"field": "GenerateRefreshToken",
		}).Error("Refresh Token generation failed!!")
	}
	return t
}

func VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			logrus.WithFields(logrus.Fields{
				"error": ok,
				"field": "VerifyToken",
			}).Error(fmt.Errorf("unexpected signing method: %v", token.Header["alg"]))
			return nil, nil
		}
		return jwtSecretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
