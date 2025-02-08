package auth

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/services"
	"github.com/sirupsen/logrus"
)

func RefreshTokenMiddleware(uc *services.UserSystem) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip token refresh for specific routes (e.g., /refresh, /login)
		if c.Path() == "/refresh" || c.Path() == "/login" || c.Path() == "/register" || fiber.RoutePatternMatch(c.Path(), "/user/:userid/activate") {
			return c.Next()
		}

		// Extract access token from cookie or header
		accessToken := c.Cookies("access_token")
		if accessToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":  "Access token missing",
				"status": fiber.StatusUnauthorized,
			})
		}

		// Validate access token
		claims, err := VerifyToken(accessToken)
		if err != nil {
			// If token is expired, attempt to refresh
			if err.Error() == "Token is expired" || err.Error() == "token has expired" || err.Error() == "token is expired" {
				return handleTokenRefresh(c, uc)
			}

			// Other errors (e.g., invalid token)
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Error("Invalid access token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":  "Unauthorized 02",
				"status": fiber.StatusUnauthorized,
			})
		}

		// Token is valid; proceed
		c.Locals("user_id", claims.UserID)
		fmt.Println("token refreshed")
		return c.Next()
	}
}

func handleTokenRefresh(c *fiber.Ctx, uc *services.UserSystem) error {
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
			"error":  "Unauthorized 03",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Fetch user from database
	userID := refreshClaims.UserID
	user, err := uc.UserBy("id = ?", userID)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":   err,
			"user_id": userID,
		}).Error("User not found")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized 04",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Generate new tokens (access + refresh)
	newAccessToken, err := GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to generate new tokens")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Token generation failed",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Set new tokens in cookies
	// Access token cookie (15 minutes)
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    newAccessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
	})

	// Update context with new access token for current request
	c.Locals("user_id", user.ID)

	// Proceed with the request
	return c.Next()
}
