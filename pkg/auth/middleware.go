package auth

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/services/users"
	"github.com/sirupsen/logrus"
)

func RefreshTokenMiddleware(uc *users.UserSystem) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip authentication for public routes
		publicRoutes := map[string]bool{
			"/login":                 true,
			"/register":              true,
			"/refresh":               true,
			"/logout":                true,
			"/user/:userid/activate": true, // Pattern matching handled below
		}
		if publicRoutes[c.Path()] || fiber.RoutePatternMatch(c.Path(), "/user/:userid/activate") {
			return c.Next()
		}

		accessToken := c.Cookies("access_token")
		if accessToken == "" {
			logger.Log.Debug("No access token found, attempting refresh")
			return handleTokenRefresh(c, uc) // Try refresh if access_token is missing but refresh_token exists
		}

		// Verify access token
		claims, err := VerifyToken(accessToken)
		if err != nil {
			if errors.Is(err, ErrExpiredToken) {
				logger.Log.Debug("Access token expired, attempting refresh")
				return handleTokenRefresh(c, uc)
			}
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Warn("Access token invalid")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":  "Invalid access token",
				"status": fiber.StatusUnauthorized,
			})
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("roles", claims.Roles)
		return c.Next()
	}
}

func handleTokenRefresh(c *fiber.Ctx, uc *users.UserSystem) error {
	// Extract refresh token from cookie
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		logger.Log.WithFields(logrus.Fields{
			"error": "Refresh token not found in cookies",
		}).Warn("Refresh token missing")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Refresh token missing, please log in again",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Validate refresh token
	refreshClaims, err := VerifyToken(refreshToken)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Refresh token invalid or expired")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Invalid or expired refresh token, please log in again",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Fetch user
	user, err := uc.UserBy("id = ?", refreshClaims.UserID)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": refreshClaims.UserID,
		}).Warn("User not found during refresh")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "User not found",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Extracting roles from User
	var rolenames []string
	for _, role := range user.Roles {
		rolenames = append(rolenames, role.Name)
	}

	// Generate new tokens (rotate refresh token for security)
	accessToken, newRefreshToken, err := GenerateJWT(*user)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": user.ID,
		}).Error("Token generation failed during refresh")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to generate new tokens",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Set new cookies with secure settings
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
		Secure:   true,     // Enforce HTTPS in production
		SameSite: "Strict", // Prevent CSRF
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	// Update context for current request
	c.Locals("user_id", user.ID)
	c.Locals("roles", refreshClaims.Roles) // Note: refresh token doesn’t have roles, so we could fetch from user instead
	logger.Log.WithFields(logrus.Fields{
		"userID": user.ID,
	}).Info("Tokens refreshed successfully")
	return c.Next()
}
