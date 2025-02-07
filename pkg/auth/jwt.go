package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/sirupsen/logrus"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
	jwtSecretKey    = []byte(os.Getenv("JWT_SECRET"))
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(userID uuid.UUID, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Minute)), // Access token expires in 15 minutes
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString(jwtSecretKey)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  email,
			"field": "GenerateAccessToken",
		}).Error("Access Token generation failed!!")
		return "", err
	}
	logger.Log.WithFields(logrus.Fields{
		"user":  email,
		"token": t,
	}).Info("Access token generated successfully")
	return t, nil
}

// GenerateRefreshToken generates a new refresh token
func GenerateRefreshToken(userID uuid.UUID) (string, error) {
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
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  userID,
			"field": "GenerateRefreshToken",
		}).Error("Refresh Token generation failed!!")
		return "", err
	}
	logger.Log.WithFields(logrus.Fields{
		"user":  userID,
		"token": t,
		"field": "GenerateRefreshToken",
	}).Info("Refresh token generated successfully!!")
	return t, nil
}

// Generate JWT Tokens
func GenerateJWT(userID uuid.UUID, email string) (string, string, error) {
	atoken, err := GenerateAccessToken(userID, email)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to generate JWT access token")
	}
	rtoken, err := GenerateRefreshToken(userID)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to generate JWT refresh token")
	}
	return atoken, rtoken, err
}

// VerifyToken verifies the token by extracting the token and cross checking secretkey, values, signing method returns
func VerifyToken(tokenString string) (*Claims, error) {
	//
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			logger.Log.WithFields(logrus.Fields{
				"error": err,
				"token": tokenString,
			}).Warn("Token has expired")
			return nil, ErrExpiredToken
		}
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"token": tokenString,
		}).Error("Invalid token")
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		logger.Log.WithFields(logrus.Fields{
			"user":  claims.Email,
			"token": tokenString,
		}).Info("Token verified successfully")
		return claims, nil
	}

	logger.Log.WithFields(logrus.Fields{
		"token": tokenString,
	}).Error("Invalid token claims")
	return nil, ErrInvalidToken
}

// ExtractToken extracts the token from the request header
func ExtractToken(c *fiber.Ctx) string {
	authHeader := c.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	logger.Log.Warn("No token found in the request header")
	return ""
}

// ExtractTokenAuth extracts and verifies the token from the request header
func ExtractTokenAuth(c *fiber.Ctx) (*Claims, error) {
	tokenString := ExtractToken(c)
	if tokenString == "" {
		logger.Log.Warn("Empty token string")
		return nil, ErrInvalidToken
	}

	return VerifyToken(tokenString)
}
