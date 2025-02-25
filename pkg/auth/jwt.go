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
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
	jwtSecretKey    = []byte(os.Getenv("JWT_SECRET"))
)

type Claims struct {
	UserID      uuid.UUID   `json:"user_id"`
	Email       string      `json:"email"`
	Roles       []string    `json:"roles"`
	Permissions []uuid.UUID `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(userID uuid.UUID, email string, roles []string, permissions []uuid.UUID) (string, error) {
	claims := Claims{
		UserID:      userID,
		Email:       email,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)), // Access token expires in 15 minutes
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "devpulse",
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString(jwtSecretKey)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"email": email,
			"field": "GenerateAccessToken",
		}).Error("Access Token generation failed!!")
		return "", fmt.Errorf("access token signing failed: %w", err)
	}
	logger.Log.WithFields(logrus.Fields{
		"email": email,
		"id":    claims.ID,
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
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "devpulse",
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString(jwtSecretKey)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
			"field":  "GenerateRefreshToken",
		}).Error("Refresh Token generation failed!!")
		return "", fmt.Errorf("refresh token signing failed: %w", err)
	}
	logger.Log.WithFields(logrus.Fields{
		"userID": userID,
		"id":     claims.ID,
		"field":  "GenerateRefreshToken",
	}).Info("Refresh token generated successfully!!")
	return t, nil
}

// Generate JWT Tokens
func GenerateJWT(user models.User, permissions []uuid.UUID) (string, string, error) {
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Name)
	}

	atoken, err := GenerateAccessToken(user.ID, user.Email, roles, permissions)
	if err != nil {
		return "", "", err
	}
	rtoken, err := GenerateRefreshToken(user.ID)
	if err != nil {
		return "", "", err
	}
	return atoken, rtoken, nil
}

// VerifyToken verifies the token by extracting the token and cross checking secretkey, values, signing method returns
func VerifyToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken // Early exit for empty tokens
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			logger.Log.WithFields(logrus.Fields{
				"alg": token.Header["alg"],
			}).Warn("Unexpected signing method")
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			logger.Log.WithFields(logrus.Fields{
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

	claims, ok := token.Claims.(*Claims)
	if !ok && !token.Valid {
		logger.Log.WithFields(logrus.Fields{
			"token": tokenString,
		}).Warn("Invalid token claims")
		return nil, ErrInvalidToken
	}

	// Additional validation: ensure UserID is present
	if claims.UserID == uuid.Nil {
		logger.Log.WithFields(logrus.Fields{
			"token": tokenString,
		}).Warn("Token missing user ID")
		return nil, ErrInvalidToken
	}

	logger.Log.WithFields(logrus.Fields{
		"userID": claims.UserID,
		"id":     claims.ID,
	}).Debug("Token verified")
	return claims, nil
}

// ExtractToken extracts the token from the request header
func ExtractToken(c *fiber.Ctx) string {
	authHeader := c.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	logger.Log.Debug("No Bearer token found in Authorization header")
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
