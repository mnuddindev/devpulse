package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/sirupsen/logrus"
)

// RoleAuth restricts access to users with specific roles
func RoleAuth(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userroles, ok := c.Locals("roles").([]string)
		if !ok {
			logger.Log.Warn("User roles not found in context")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":  "User roles unavailable",
				"status": fiber.StatusForbidden,
			})
		}

		// Allow all authenticated users if "all" is specified
		for _, role := range roles {
			if role == "all" {
				return c.Next()
			}
		}

		// Check for role match
		for _, requiredRole := range roles {
			for _, userRole := range userroles {
				if userRole == requiredRole {
					logger.Log.WithFields(logrus.Fields{
						"userID": c.Locals("user_id"),
						"role":   userRole,
					}).Debug("Role authorized")
					return c.Next()
				}
			}
		}

		logger.Log.WithFields(logrus.Fields{
			"userID":     c.Locals("user_id"),
			"required":   roles,
			"user_roles": userroles,
		}).Warn("Insufficient permissions")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":  "Insufficient permissions",
			"status": fiber.StatusForbidden,
		})
	}
}
