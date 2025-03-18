package auth

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/internal/models"
	"github.com/mnuddindev/devpulse/pkg/utils"
)

func CheckPerm(opt Options, perms ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var perm []string
		if len(perms) > 0 && perms[0] != "" {
			for _, p := range perms {
				perm = append(perm, p)
			}
		}

		user_id := c.Locals("user_id").(string)
		userKey := "user:" + user_id
		var user *models.User
		cachedUser, err := opt.Rclient.Get(c.Context(), userKey).Result()
		if err == nil && cachedUser != "" {
			user = &models.User{}
			if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
				opt.Logger.Warn(c.Context()).WithFields("error", err).Logs("Failed to unmarshal cached user")
				user = nil
			}
		}
		if err != nil {
			opt.Logger.Warn(c.Context()).WithFields("user_id", user_id).Logs("Redis error fetching user")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		permissions := GetPermissions(user)

		hasPermission := false
		for _, required := range perm {
			ok := utils.Contains(permissions, required)
			if !ok {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			opt.Logger.Warn(c.Context()).WithFields(
				"user_id", c.Locals("user_id").(string),
				"required_perms", perm,
				"user_permissions", permissions,
			).Logs("Insufficient permissions")
			// Return a 403 Forbidden response if the non-admin user lacks all required permissions
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":  "Insufficient permissions",
				"status": fiber.StatusForbidden,
			})
		}

		opt.Logger.Debug(c.Context()).WithFields("user_id", user_id).Logs("Permission authorized")
		return c.Next()
	}
}

// GetPermissions extracts permission names into a []string
func GetPermissions(u *models.User) []string {
	if u == nil || len(u.Role.Permissions) == 0 {
		return []string{}
	}
	permissions := make([]string, len(u.Role.Permissions))
	for i, perm := range u.Role.Permissions {
		permissions[i] = perm.Name
	}
	return permissions
}
