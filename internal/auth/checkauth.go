package auth

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/internal/models"
)

func CheckPerm(opt Options, perms ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
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

		permMap := make(map[string]bool)
		for _, p := range perms {
			permMap[p] = true
		}

		hasPermission, err := GetPermissions(c, opt, user.RoleID, permMap)
		if err != nil {
			opt.Logger.Warn(c.Context()).WithFields("error", err).Logs("Failed to get permissions")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal Server Error",
			})
		}

		if !hasPermission {
			opt.Logger.Warn(c.Context()).WithFields(
				"user_id", c.Locals("user_id").(string),
				"required_perms", perms,
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
func GetPermissions(c *fiber.Ctx, opt Options, roleid uuid.UUID, permName map[string]bool) (bool, error) {
	key := "role_perms:" + roleid.String()

	cachedPerms, err := opt.Rclient.Get(c.Context(), key).Result()
	if err == nil && cachedPerms != "" {
		var permissions []string
		if err := json.Unmarshal([]byte(cachedPerms), &permissions); err != nil {
			opt.Logger.Warn(c.Context()).WithFields("error", err, "role_id", roleid).Logs("Failed to unmarshal cached permissions")
			permissions = nil
		} else {
			for _, perm := range permissions {
				if permName[perm] {
					return true, nil
				}
			}
			return false, nil
		}
	}

	// Fall back role
	var role models.Role
	if err := opt.DB.WithContext(c.Context()).Preload("Permissions").Where("id = ?", roleid).First(&role).Error; err != nil {
		opt.Logger.Warn(c.Context()).WithFields("error", err).WithFields("role_id", roleid).Logs("Role not found in DB")
		return false, err
	}

	for _, perm := range role.Permissions {
		if permName[perm.Name] {
			// Cache in Redis
			permNames := make([]string, len(role.Permissions))
			for i, p := range role.Permissions {
				permNames[i] = p.Name
			}
			permJSON, _ := json.Marshal(permNames)
			opt.Rclient.Set(c.Context(), key, permJSON, 24*time.Hour)
			return true, nil
		}
	}

	return false, nil
}
