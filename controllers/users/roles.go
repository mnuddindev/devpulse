package users

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/sirupsen/logrus"
)

// AddRoleToUser assigns a role to a user (CREATE)
func (uc *UserController) AddRoleToUser(c *fiber.Ctx) error {
	// Define a struct to parse the JSON request body
	type Request struct {
		UserID uuid.UUID `json:"user_id" validate:"required"`
		RoleID uuid.UUID `json:"role_id" validate:"required"`
	}

	// Declare a variable to hold the parsed request data
	var req Request
	// Parse the JSON request body into the Request struct
	if err := utils.StrictBodyParser(c, &req); err != nil {
		// Return a 400 Bad Request response if parsing fails (e.g., invalid JSON)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Validate input
	if req.UserID == uuid.Nil || req.RoleID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User ID and Role ID are required"})
	}

	// Query the database for the user by ID, preloading their current roles
	user, err := uc.userSystem.UserBy("id = ?", req.UserID)
	if err != nil {
		// Return a 404 Not Found response if the user doesn’t exist
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Query the database for the role by ID
	role, err := uc.userSystem.RoleBy("id = ?", req.RoleID)
	if err != nil {
		// Return a 404 Not Found response if the role doesn’t exist
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
	}

	// Add the role to the user’s list of roles in the many-to-many relationship
	if err := uc.userSystem.Crud.AddManyToMany(&user, "Roles", &role); err != nil {
		// Log an error if the database operation fails
		logger.Log.WithError(err).Error("Failed to assign role")
		// Return a 500 Internal Server Error response if the operation fails
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to assign role"})
	}

	// Log an info message confirming the role assignment
	logger.Log.WithFields(logrus.Fields{
		"user_id": req.UserID,
		"role":    role.Name,
	}).Info("Role assigned to user")
	// Return a 200 OK response with a success message
	return c.JSON(fiber.Map{"message": "Role assigned successfully"})
}

// GetUserPermissions retrieves a user's permissions (READ)
func (uc *UserController) GetUserPermissions(c *fiber.Ctx) error {
	// Get the user ID from the URL parameter
	userID := c.Params("user_id")
	// Attempt to parse the user ID as a UUID to validate its format
	if _, err := uuid.Parse(userID); err != nil {
		// Return a 400 Bad Request response if the user ID is not a valid UUID
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// Query the database for the user by ID, preloading roles and their permissions
	user, err := uc.userSystem.UserBy("id = ?", userID)
	if err != nil {
		// Return a 404 Not Found response if the user doesn’t exist
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Create a map to store unique permissions (using a map to avoid duplicates)
	permissions := make(map[string]bool)
	// Iterate over the user’s roles
	for _, role := range user.Roles {
		// Iterate over the permissions associated with each role
		for _, perm := range role.Permissions {
			// Add the permission name to the map with a true value
			permissions[perm.Name] = true
		}
	}

	// Return a 200 OK response with the user’s ID, username, and permissions
	return c.JSON(fiber.Map{
		"user_id":     user.ID,
		"username":    user.Username,
		"permissions": permissions,
	})
}

// UpdateUserRolePermissions modifies a user's role permissions (UPDATE)
func (uc *UserController) UpdateUserRolePermissions(c *fiber.Ctx) error {
	// Define a struct to parse the JSON request body
	type Request struct {
		UserID      uuid.UUID `json:"user_id" validate:"required"`
		RoleID      uuid.UUID `json:"role_id" validate:"required"`
		Permissions []string  `json:"permissions" validate:"required"`
	}

	// Declare a variable to hold the parsed request data
	var req Request
	// Parse the JSON request body into the Request struct
	if err := utils.StrictBodyParser(c, &req); err != nil {
		// Return a 400 Bad Request response if parsing fails
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Query the database for the user by ID, preloading their current roles
	_, err := uc.userSystem.UserBy("id = ?", req.UserID)
	if err != nil {
		// Return a 404 Not Found response if the user doesn’t exist
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Query the database for the role by ID, preloading its current permissions
	role, err := uc.userSystem.RoleBy("id = ?", req.RoleID)
	if err != nil {
		// Return a 404 Not Found response if the role doesn’t exist
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
	}

	// Declare a slice to hold the fetched permissions from the database
	var perms []models.Permission
	// Query the database for permissions matching the names provided in the request
	if err := uc.DB.Where("name IN ?", req.Permissions).Find(&perms).Error; err != nil || len(perms) != len(req.Permissions) {
		// Return a 400 Bad Request response if any permission names are invalid or not found
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid permissions"})
	}

	// Replace the role’s current permissions with the new set of permissions
	if err := uc.userSystem.Crud.UpdateManyToMany(&role, "Permissions", &perms); err != nil {
		// Log an error if the database operation fails
		logrus.WithError(err).Error("Failed to update permissions")
		// Return a 500 Internal Server Error response if the operation fails
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update permissions"})
	}

	// Log an info message confirming the permission update
	logrus.WithFields(logrus.Fields{
		"user_id":     req.UserID,
		"role":        role.Name,
		"permissions": req.Permissions,
	}).Info("User role permissions updated")
	// Return a 200 OK response with a success message
	return c.JSON(fiber.Map{"message": "Permissions updated successfully"})
}

// RemoveRoleFromUser deletes a role from a user (DELETE)
func (uc *UserController) RemoveRoleFromUser(c *fiber.Ctx) error {
	// Define a struct to parse the JSON request body
	type Request struct {
		UserID uuid.UUID `json:"user_id" validate:"required"`
		RoleID uuid.UUID `json:"role_id" validate:"required"`
	}

	// Declare a variable to hold the parsed request data
	var req Request
	// Parse the JSON request body into the Request struct
	if err := c.BodyParser(&req); err != nil {
		// Return a 400 Bad Request response if parsing fails
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	user, err := uc.userSystem.UserBy("id = ?", req.UserID)
	if err != nil {
		// Return a 404 Not Found response if the user doesn’t exist
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Query the database for the role by ID, preloading its current permissions
	role, err := uc.userSystem.RoleBy("id = ?", req.RoleID)
	if err != nil {
		// Return a 404 Not Found response if the role doesn’t exist
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
	}

	// Remove the specified role from the user’s list of roles
	if err := uc.userSystem.Crud.DeleteManyToMany(&user, "Roles", &role); err != nil {
		// Log an error if the database operation fails
		logrus.WithError(err).Error("Failed to remove role")
		// Return a 500 Internal Server Error response if the operation fails
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to remove role"})
	}

	// Log an info message confirming the role removal
	logrus.WithFields(logrus.Fields{
		"user_id": req.UserID,
		"role":    role.Name,
	}).Info("Role removed from user")
	// Return a 200 OK response with a success message
	return c.JSON(fiber.Map{"message": "Role removed successfully"})
}
