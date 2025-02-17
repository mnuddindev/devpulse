package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/services"
	"github.com/sirupsen/logrus"
)

func IsAuth(uc *services.UserSystem) fiber.Handler {
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
				if err := handleTokenRefresh(c, uc); err != nil {
					return err
				}

				// Retry extracting claims after refresh
				newAccessToken := c.Cookies("access_token")
				claims, err = VerifyToken(newAccessToken)
				if err != nil {
					logger.Log.WithFields(logrus.Fields{
						"error": err,
					}).Error("Invalid refreshed token")
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error":  "Unauthorized",
						"status": fiber.StatusUnauthorized,
					})
				}
			}
		}

		// Extract roles
		roles := claims.Roles

		// Attach user ID to context
		c.Locals("user_id", claims.UserID)
		c.Locals("roles", roles)

		return c.Next()
	}
}
