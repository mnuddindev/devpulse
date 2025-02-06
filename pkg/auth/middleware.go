package auth

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/services"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var uc *services.UserSystem

func RefreshTokenMiddleware(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip token refresh for specific routes (e.g., /refresh, /login)
		if c.Path() == "/refresh" || c.Path() == "/login" || c.Path() == "/register" {
			return c.Next()
		}

		// Extract access token from cookie or header
		accessToken := c.Cookies("access_token")
		if accessToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":  "Unauthorized",
				"status": fiber.StatusUnauthorized,
			})
		}

		// Validate access token
		claims, err := VerifyToken(accessToken)
		if err != nil {
			// If token is expired, attempt to refresh
			if err.Error() == "Token is expired" {
				return handleTokenRefresh(c)
			}

			// Other errors (e.g., invalid token)
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Error("Invalid access token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":  "Unauthorized",
				"status": fiber.StatusUnauthorized,
			})
		}

		// Token is valid; proceed
		c.Locals("user_id", claims.UserID)
		fmt.Println("token refreshed")
		return c.Next()
	}
}

func handleTokenRefresh(c *fiber.Ctx) error {
	// Extract refresh token from cookie
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Refresh token missing",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Validate refresh token
	refreshClaims, err := VerifyToken(refreshToken)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Invalid refresh token")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Fetch user from database
	userID := refreshClaims.UserID
	user, err := uc.UserBy("id = ?", userID)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("User not found")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Generate new tokens (access + refresh)
	newAccessToken, newRefreshToken, err := GenerateJWT(user.ID, user.Email)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to generate new tokens")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Internal server error",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Set new tokens in cookies
	setTokenCookies(c, newAccessToken, newRefreshToken)

	// Update context with new access token for current request
	c.Locals("user_id", user.ID.String())

	// Proceed with the request
	return c.Next()
}

func setTokenCookies(c *fiber.Ctx, accessToken, refreshToken string) {
	// Access token cookie (15 minutes)
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
		Secure:   true,     // Use in production (HTTPS-only)
		SameSite: "Strict", // Prevent CSRF
	})

	// Refresh token cookie (30 days)
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   true,     // Use in production (HTTPS-only)
		SameSite: "Strict", // Prevent CSRF
	})
}
