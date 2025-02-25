package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// PermissionAuth restricts access to users with specific permissions
func PermissionAuth(db *gorm.DB, requiredPerms ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Attempt to retrieve permissions from the context, expecting a []string
		perms, ok := c.Locals("permissions").([]string)
		// Check if permissions are not found or not in the expected type
		if !ok {
			// Retrieve the user ID from the context, assuming it’s a string set by earlier middleware
			userID := c.Locals("user_id").(string)
			// Declare a User variable to hold the fetched user data from the database
			var user models.User
			// Query the database for the user by ID, preloading roles and their permissions
			if err := db.Preload("Roles.Permissions").Where("id = ?", userID).First(&user).Error; err != nil {
				// Log a warning if the user isn’t found in the database
				logrus.Warn("User not found in database fallback")
				// Return a 403 Forbidden response if the user doesn’t exist
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "User not found"})
			}
			// Extract permissions from the user’s roles using the helper function
			perms = getUserPermissions(&user)
			// Store the fetched permissions back in the context for future use in this request
			c.Locals("permissions", perms)
		}

		// Iterate over the required permissions passed to the middleware
		for _, perm := range requiredPerms {
			// Check if "all" is specified, allowing any authenticated user to proceed
			if perm == "all" {
				// Proceed to the next handler without further checks
				return c.Next()
			}
		}

		// Create a map to efficiently check for permission existence
		permMap := make(map[string]bool)
		// Populate the map with the user’s permissions
		for _, p := range perms {
			permMap[p] = true
		}
		// Iterate over the required permissions to verify each one
		for _, required := range requiredPerms {
			// Check if the required permission is missing from the user’s permissions
			if !permMap[required] {
				// Log a warning with details about the missing permissions
				logrus.WithFields(logrus.Fields{
					"user_id":          c.Locals("user_id"),
					"required_perms":   requiredPerms,
					"user_permissions": perms,
				}).Warn("Insufficient permissions")
				// Return a 403 Forbidden response if any required permission is missing
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":  "Insufficient permissions",
					"status": fiber.StatusForbidden,
				})
			}
		}

		// Log a debug message confirming the user has all required permissions
		logrus.WithFields(logrus.Fields{
			"user_id":     c.Locals("user_id"),
			"permissions": perms,
		}).Debug("Permission authorized")
		// Proceed to the next handler since all permissions are satisfied
		return c.Next()
	}
}

// getUserPermissions extracts permissions from a user’s roles
func getUserPermissions(user *models.User) []string {
	// Initialize an empty slice to store the user’s permissions
	var perms []string
	// Iterate over the user’s roles
	for _, role := range user.Roles {
		// Iterate over the permissions associated with each role
		for _, perm := range role.Permissions {
			// Append each permission name to the slice
			perms = append(perms, perm.Name)
		}
	}
	// Return the list of permission names
	return perms
}
