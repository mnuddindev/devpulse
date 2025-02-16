package auth

import "github.com/gofiber/fiber/v2"

func RoleAuth(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userroles, ok := c.Locals("roles").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "User roles not found",
			})
		}

		// If "all" is passed, allow all users with any role
		for _, role := range roles {
			if role == "all" {
				return c.Next()
			}
		}

		// Check if user has at least one of the required roles
		for _, role := range roles {
			for _, userRole := range userroles {
				if userRole == role {
					return c.Next()
				}
			}
		}

		// If no matching role is found, return forbidden
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient Permission",
		})
	}
}
