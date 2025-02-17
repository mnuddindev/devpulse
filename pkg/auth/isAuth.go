package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/sirupsen/logrus"
)

func IsAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		accessToken := c.Cookies("access_token")
		if accessToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":  "Access token missing",
				"status": fiber.StatusUnauthorized,
			})
		}

		// Verify token
		claims, err := VerifyToken(accessToken)
		if err != nil {
			if errors.Is(err, ErrExpiredToken) {
				logger.Log.WithFields(logrus.Fields{
					"error": "Expired token",
				}).Warn("Access denied: Token expired")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":  "Token expired. Please log in again.",
					"status": fiber.StatusUnauthorized,
				})
			}

			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Error("Invalid token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":  "Invalid token",
				"status": fiber.StatusUnauthorized,
			})
		}

		// Extract roles
		roles := claims.Roles

		// Attach user ID to context
		c.Locals("user_id", claims.UserID)
		c.Locals("roles", roles)

		// Secure cookie settings
		c.Cookie(&fiber.Cookie{
			Name:     "access_token",
			Value:    accessToken,
			Secure:   true,
			HTTPOnly: true,
			SameSite: "Strict",
		})

		return c.Next()
	}
}
