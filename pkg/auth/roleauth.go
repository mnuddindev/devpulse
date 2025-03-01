package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// PermissionAuth restricts access to users with specific permissions, allowing admins full access
func PermissionAuth(db *gorm.DB, requiredPerms ...string) fiber.Handler {
	// Return a Fiber handler function that processes the permission check
	return func(c *fiber.Ctx) error {
		// Attempt to retrieve permissions from the context, expecting a []string type set by prior middleware
		perms, ok := c.Locals("permissions").([]string)
		// Check if permissions are not found in the context or not in the expected type
		if !ok {
			// Retrieve the user ID from the context, assuming it’s a string set by earlier middleware (e.g., JWT auth)
			userID := c.Locals("user_id").(string)
			// Declare a User variable to hold the fetched user data from the database
			var user models.User
			// Query the database for the user by ID, preloading roles and their permissions for a complete check
			if err := db.Preload("Roles.Permissions").Where("id = ?", userID).First(&user).Error; err != nil {
				// Log a warning if the user isn’t found in the database during the fallback check
				logrus.Warn("User not found in database fallback")
				// Return a 403 Forbidden response if the user doesn’t exist, preventing further processing
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "User not found"})
			}
			// Extract permissions from the user’s roles using the helper function
			perms = getUserPermissions(&user)
			// Store the fetched permissions back in the context for use in subsequent handlers
			c.Locals("permissions", perms)
		}

		// Create a map to efficiently check for permission existence among the user’s permissions
		permMap := make(map[string]bool)
		// Populate the map with the user’s permissions for quick lookup
		for _, p := range perms {
			permMap[p] = true
		}

		// Check if the user has the "admin" permission, granting unrestricted access
		if permMap["admin"] {
			// Log a debug message confirming the admin user has full access
			logrus.WithFields(logrus.Fields{
				"user_id":     c.Locals("user_id"),
				"permissions": perms,
			}).Debug("Admin permission authorized, granting full access")
			// Proceed to the next handler immediately, bypassing further permission checks
			return c.Next()
		}

		// Iterate over special permissions like "all" to allow immediate access if present (non-admin case)
		for _, perm := range requiredPerms {
			// Check if "all" is specified, allowing any authenticated non-admin user to proceed
			if perm == "all" {
				// Proceed to the next handler without additional permission validation
				return c.Next()
			}
		}

		// Check if the user has at least one of the required permissions for non-admins
		hasPermission := false
		// Iterate over the required permissions to find a match among non-admin users
		for _, required := range requiredPerms {
			// If the user has the required permission, set the flag to true
			if permMap[required] {
				hasPermission = true
				// Break the loop since one match is sufficient for OR logic
				break
			}
		}
		// Check if no required permissions were found in the user’s set for non-admins
		if !hasPermission {
			// Log a warning with details about the missing permissions for audit purposes
			logrus.WithFields(logrus.Fields{
				"user_id":          c.Locals("user_id"),
				"required_perms":   requiredPerms,
				"user_permissions": perms,
			}).Warn("Insufficient permissions")
			// Return a 403 Forbidden response if the non-admin user lacks all required permissions
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":  "Insufficient permissions",
				"status": fiber.StatusForbidden,
			})
		}

		// Log a debug message confirming the non-admin user has at least one required permission
		logrus.WithFields(logrus.Fields{
			"user_id":     c.Locals("user_id"),
			"permissions": perms,
		}).Debug("Permission authorized")
		// Proceed to the next handler since the non-admin user has at least one required permission
		return c.Next()
	}
}

// getUserPermissions extracts permissions from a user’s roles (unchanged from your original)
func getUserPermissions(user *models.User) []string {
	// Initialize an empty slice to store the user’s permissions
	var perms []string
	// Iterate over the user’s roles to access their associated permissions
	for _, role := range user.Roles {
		// Iterate over the permissions within each role
		for _, perm := range role.Permissions {
			// Append each permission name to the slice
			perms = append(perms, perm.Name)
		}
	}
	// Return the collected list of permission names
	return perms
}
