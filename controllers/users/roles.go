package users

import (
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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

// CreateRole adds a new role to the system, restricted to admins with specific permissions
func (uc *UserController) CreateRole(c *fiber.Ctx) error {
	// Attempt to retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is authenticated
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized role creation attempts without authentication
		logger.Log.Warn("CreateRole attempted without authenticated admin ID")
		// Return a 401 Unauthorized response with a structured error; PermissionAuth should catch this earlier
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth", Msg: "Admin authentication required"}},
			"status": fiber.StatusUnauthorized,
		})
	}

	// Parse the authenticated admin user ID into a UUID for rate limiting and logging
	authUserID, err := uuid.Parse(authUserIDRaw)
	// Check if parsing the authenticated admin user ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid authenticated admin ID formats from the context
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserIDRaw,
		}).Error("Invalid authenticated admin ID format in CreateRole")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting admin role creation attempts, unique to the authenticated admin’s UUID
	rateKey := "create_role_rate:" + authUserID.String()
	// Define a constant for the maximum number of role creation attempts allowed per hour by an admin
	const maxAttempts = 5
	// Retrieve the current count of role creation attempts from Redis for this admin user
	attemptsCount, err := uc.Client.Get(c.Context(), rateKey).Int()
	// Check if the rate limit key doesn’t exist in Redis, indicating the first attempt in the time window
	if err == redis.Nil {
		// Set the attempt count to 0 since no previous attempts have been recorded for this admin
		attemptsCount = 0
		// Check if an unexpected Redis error occurred beyond a missing key
	} else if err != nil {
		// Log a warning to monitor Redis connectivity or configuration issues, but proceed without blocking
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserID,
		}).Warn("Failed to check role creation rate limit in Redis")
	}
	// Check if the number of role creation attempts exceeds the allowed maximum for this admin
	if attemptsCount >= maxAttempts {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Role creation rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive attempts by this admin
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "create_role", Msg: "Too many role creation attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the attempt counter in Redis to record this role creation attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-hour expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Hour)

	// Define a struct to parse the JSON request body containing role details, with validation rules
	type Request struct {
		// Define Name as a required string field for the role’s name, matching model’s size constraint
		Name string `json:"name" validate:"required,min=2,max=50"`
	}

	// Allocate a new Request struct to store the parsed request body data from the admin
	req := new(Request)
	// Parse the request body strictly into the Request struct, ensuring it matches the expected JSON structure
	if err := utils.StrictBodyParser(c, req); err != nil {
		// Log an error with details to debug parsing issues with the request body submitted by the admin
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body in CreateRole")
		// Return a 400 Bad Request response with a structured error indicating an invalid body
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "request", Msg: "Invalid request body"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Create a new validator instance to enforce validation rules defined in the Request struct
	validator := utils.NewValidator()
	// Validate the Request struct to ensure the Name field is present and meets length constraints
	validationErrors := validator.Validate(req)
	// Check if validation returned any errors, indicated by a non-nil ErrorResponse with non-empty Errors
	if validationErrors != nil && len(validationErrors.Errors) > 0 {
		// Log an error with details about validation failures to assist in debugging admin input issues
		logger.Log.WithFields(logrus.Fields{
			"error": validationErrors,
		}).Error("Validation failed in CreateRole")
		// Return a 422 Unprocessable Entity response with the structured validation errors for admin correction
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": validationErrors.Errors,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Create a new Role instance with the provided name and a generated UUID
	role := models.Role{
		ID:   uuid.New(),
		Name: req.Name,
	}
	// Attempt to create the role in the database using the Crud system
	if err := uc.userSystem.Crud.Create(&role); err != nil {
		// Log an error with details if the database operation to create the role fails (e.g., unique constraint violation)
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"name":  req.Name,
		}).Error("Failed to create role")
		// Check if the error is due to a unique constraint violation (e.g., duplicate name)
		if err.Error() == "duplicate key value violates unique constraint" {
			// Return a 409 Conflict response with a structured error indicating the role name already exists
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "name", Msg: "Role name already exists"}},
				"status": fiber.StatusConflict,
			})
		}
		// Return a 500 Internal Server Error for other database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "role_creation", Msg: "Failed to create role"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Log a success message with details to confirm the role was created by the admin
	logger.Log.WithFields(logrus.Fields{
		"roleID":      role.ID,
		"name":        role.Name,
		"adminUserID": authUserID,
	}).Info("Role created successfully by admin")
	// Return a 201 Created response with the role ID and a success message for the admin
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Role created successfully",
		"role_id": role.ID,
		"status":  fiber.StatusCreated,
	})
}

// GetRole retrieves details of a specific role, accessible by admins
func (uc *UserController) GetRole(c *fiber.Ctx) error {
	// Attempt to retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is authenticated
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized role retrieval attempts without authentication
		logger.Log.Warn("GetRole attempted without authenticated admin ID")
		// Return a 401 Unauthorized response with a structured error; PermissionAuth should catch this earlier
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth", Msg: "Admin authentication required"}},
			"status": fiber.StatusUnauthorized,
		})
	}

	// Parse the authenticated admin user ID into a UUID for rate limiting and logging
	authUserID, err := uuid.Parse(authUserIDRaw)
	// Check if parsing the authenticated admin user ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid authenticated admin ID formats from the context
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserIDRaw,
		}).Error("Invalid authenticated admin ID format in GetRole")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Get the target role ID from the URL parameter "role_id"
	roleIDRaw := c.Params("role_id")
	// Attempt to parse the target role ID as a UUID to validate its format
	roleID, err := uuid.Parse(roleIDRaw)
	// Check if parsing the target role ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid role ID formats provided in the request URL
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": roleIDRaw,
		}).Error("Invalid role ID format in GetRole")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "role_id", Msg: "Invalid role ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting role retrieval attempts, unique to the authenticated admin’s UUID
	rateKey := "get_role_rate:" + authUserID.String()
	// Define a constant for the maximum number of role retrieval attempts allowed per minute
	const maxAttempts = 100
	// Retrieve the current count of role retrieval attempts from Redis for this admin user
	attemptsCount, err := uc.Client.Get(c.Context(), rateKey).Int()
	// Check if the rate limit key doesn’t exist in Redis, indicating the first attempt in the time window
	if err == redis.Nil {
		// Set the attempt count to 0 since no previous attempts have been recorded
		attemptsCount = 0
		// Check if an unexpected Redis error occurred beyond a missing key
	} else if err != nil {
		// Log a warning to monitor Redis connectivity or configuration issues, but proceed without blocking
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserID,
		}).Warn("Failed to check role retrieval rate limit in Redis")
	}
	// Check if the number of role retrieval attempts exceeds the allowed maximum
	if attemptsCount >= maxAttempts {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Role retrieval rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive attempts
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "get_role", Msg: "Too many role retrieval attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the attempt counter in Redis to record this role retrieval attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-minute expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Minute)

	// Generate a Redis key for caching the role data, using the target role’s UUID
	roleKey := "role:" + roleID.String()
	// Attempt to fetch the role data from Redis for a fast response
	cachedRole, err := uc.Client.Get(c.Context(), roleKey).Result()
	// Declare a variable to hold the role struct, which will be populated from Redis or DB
	var role *models.Role
	// Check if the role data was successfully retrieved from Redis without errors
	if err == nil {
		// Allocate a new Role struct to unmarshal the cached data into
		role = &models.Role{}
		// Unmarshal the JSON data fetched from Redis into the role struct
		if err := json.Unmarshal([]byte(cachedRole), role); err != nil {
			// Log a warning if unmarshaling fails, indicating potential corruption in the cached data
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"roleID": roleID,
			}).Warn("Failed to unmarshal cached role from Redis in GetRole")
			// Set role to nil to force a database lookup if the cached data is unusable
			role = nil
		}
	}
	// Check if the role wasn’t found in Redis (cache miss) or if unmarshaling failed
	if err == redis.Nil || role == nil {
		// Fetch the role from the database using the provided role ID, preloading permissions
		role, err = uc.userSystem.RoleBy("id = ?", roleID)
		// Check if fetching the role failed
		if err != nil {
			// Log an error with details to debug database retrieval issues for the admin
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"roleID": roleID,
			}).Error("Failed to fetch role")
			// Check if the error indicates the role doesn’t exist in the database
			if err == gorm.ErrRecordNotFound {
				// Return a 404 Not Found response with a structured error indicating the role wasn’t found
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"errors": []utils.Error{{Field: "role_id", Msg: "Role not found"}},
					"status": fiber.StatusNotFound,
				})
			}
			// Return a 500 Internal Server Error for unexpected database errors
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
				"status": fiber.StatusInternalServerError,
			})
		}
		// Marshal the role data into JSON for caching
		roleJSON, err := json.Marshal(role)
		// Check if marshaling the role data failed
		if err != nil {
			// Log a warning if marshaling fails, but proceed since we have the role data
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"roleID": roleID,
			}).Warn("Failed to marshal role for Redis caching in GetRole")
		} else {
			// Cache the role data in Redis with a 30-minute TTL for quick future access
			if err := uc.Client.Set(c.Context(), roleKey, roleJSON, 30*time.Minute).Err(); err != nil {
				// Log a warning if caching fails, but proceed as DB data is authoritative
				logger.Log.WithFields(logrus.Fields{
					"error":  err,
					"roleID": roleID,
				}).Warn("Failed to cache role in Redis")
			}
		}
	}

	// Extract permissions from the role for the response, aligning with the many-to-many relationship
	permissions := make([]string, len(role.Permissions))
	// Iterate over the role’s permissions to populate the permissions list
	for i, perm := range role.Permissions {
		// Add each permission name to the slice
		permissions[i] = perm.Name
	}

	// Log a success message with details to confirm the role was retrieved by the admin
	logger.Log.WithFields(logrus.Fields{
		"roleID":      roleID,
		"name":        role.Name,
		"adminUserID": authUserID,
	}).Debug("Role retrieved successfully by admin")
	// Return a 200 OK response with the role details and permissions for the admin
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Role retrieved successfully",
		"status":  fiber.StatusOK,
		"role": fiber.Map{
			"id":          role.ID,
			"name":        role.Name,
			"permissions": permissions,
			"created_at":  role.CreatedAt,
			"updated_at":  role.UpdatedAt,
		},
	})
}

// ListRoles retrieves a list of all roles, accessible by admins
func (uc *UserController) ListRoles(c *fiber.Ctx) error {
	// Attempt to retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is authenticated
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized role list retrieval attempts without authentication
		logger.Log.Warn("ListRoles attempted without authenticated admin ID")
		// Return a 401 Unauthorized response with a structured error; PermissionAuth should catch this earlier
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth", Msg: "Admin authentication required"}},
			"status": fiber.StatusUnauthorized,
		})
	}

	// Parse the authenticated admin user ID into a UUID for rate limiting and logging
	authUserID, err := uuid.Parse(authUserIDRaw)
	// Check if parsing the authenticated admin user ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid authenticated admin ID formats from the context
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserIDRaw,
		}).Error("Invalid authenticated admin ID format in ListRoles")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting role list retrieval attempts, unique to the authenticated admin’s UUID
	rateKey := "list_roles_rate:" + authUserID.String()
	// Define a constant for the maximum number of role list retrieval attempts allowed per minute
	const maxAttempts = 100
	// Retrieve the current count of role list retrieval attempts from Redis for this admin user
	attemptsCount, err := uc.Client.Get(c.Context(), rateKey).Int()
	// Check if the rate limit key doesn’t exist in Redis, indicating the first attempt in the time window
	if err == redis.Nil {
		// Set the attempt count to 0 since no previous attempts have been recorded
		attemptsCount = 0
		// Check if an unexpected Redis error occurred beyond a missing key
	} else if err != nil {
		// Log a warning to monitor Redis connectivity or configuration issues, but proceed without blocking
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserID,
		}).Warn("Failed to check role list retrieval rate limit in Redis")
	}
	// Check if the number of role list retrieval attempts exceeds the allowed maximum
	if attemptsCount >= maxAttempts {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Role list retrieval rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive attempts
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "list_roles", Msg: "Too many role list retrieval attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the attempt counter in Redis to record this role list retrieval attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-minute expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Minute)

	// Generate a Redis key for caching the list of all roles
	rolesKey := "roles:all"
	// Attempt to fetch the cached list of roles from Redis for a fast response
	cachedRoles, err := uc.Client.Get(c.Context(), rolesKey).Result()
	// Declare a variable to hold the slice of roles
	var roles []models.Role
	// Check if the roles data was successfully retrieved from Redis without errors
	if err == nil {
		// Unmarshal the cached JSON data into the roles slice
		if err := json.Unmarshal([]byte(cachedRoles), &roles); err != nil {
			// Log a warning if unmarshaling fails, indicating potential corruption in the cached data
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Warn("Failed to unmarshal cached roles from Redis in ListRoles")
			// Set roles to nil to force a database lookup if the cached data is unusable
			roles = nil
		}
	}
	// Check if the roles weren’t found in Redis (cache miss) or if unmarshaling failed
	if err == redis.Nil || roles == nil {
		// Fetch all roles from the database; since no DeletedAt field exists, assume active roles only or Crud handles it
		if err := uc.DB.Find(&roles).Error; err != nil {
			// Log an error with details to debug database retrieval issues for the admin
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Error("Failed to fetch roles from database")
			// Return a 500 Internal Server Error for unexpected database errors
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
				"status": fiber.StatusInternalServerError,
			})
		}
		// Marshal the roles slice into JSON for caching
		rolesJSON, err := json.Marshal(roles)
		// Check if marshaling the roles failed
		if err != nil {
			// Log a warning if marshaling fails, but proceed since we have the roles data
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Warn("Failed to marshal roles for Redis caching in ListRoles")
		} else {
			// Cache the roles list in Redis with a 30-minute TTL for quick future access
			if err := uc.Client.Set(c.Context(), rolesKey, rolesJSON, 30*time.Minute).Err(); err != nil {
				// Log a warning if caching fails, but proceed as DB data is authoritative
				logger.Log.WithFields(logrus.Fields{
					"error": err,
				}).Warn("Failed to cache roles in Redis")
			}
		}
	}

	// Prepare a slice to hold simplified role data for the response
	roleList := make([]fiber.Map, len(roles))
	// Iterate over the roles to populate the response data
	for i, role := range roles {
		// Create a map for each role with its ID, name, created_at, and updated_at fields (no Description)
		roleList[i] = fiber.Map{
			"id":         role.ID,
			"name":       role.Name,
			"created_at": role.CreatedAt,
			"updated_at": role.UpdatedAt,
		}
	}

	// Log a success message with details to confirm the roles were retrieved by the admin
	logger.Log.WithFields(logrus.Fields{
		"adminUserID": authUserID,
		"role_count":  len(roles),
	}).Debug("Roles listed successfully by admin")
	// Return a 200 OK response with the list of roles for the admin
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Roles retrieved successfully",
		"status":  fiber.StatusOK,
		"roles":   roleList,
	})
}

// UpdateRole modifies an existing role’s name, restricted to admins
func (uc *UserController) UpdateRole(c *fiber.Ctx) error {
	// Attempt to retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is authenticated
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized role update attempts without authentication
		logger.Log.Warn("UpdateRole attempted without authenticated admin ID")
		// Return a 401 Unauthorized response with a structured error; PermissionAuth should catch this earlier
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth", Msg: "Admin authentication required"}},
			"status": fiber.StatusUnauthorized,
		})
	}

	// Parse the authenticated admin user ID into a UUID for rate limiting and logging
	authUserID, err := uuid.Parse(authUserIDRaw)
	// Check if parsing the authenticated admin user ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid authenticated admin ID formats from the context
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserIDRaw,
		}).Error("Invalid authenticated admin ID format in UpdateRole")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Get the target role ID from the URL parameter "role_id"
	roleIDRaw := c.Params("role_id")
	// Attempt to parse the target role ID as a UUID to validate its format
	roleID, err := uuid.Parse(roleIDRaw)
	// Check if parsing the target role ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid role ID formats provided in the request URL
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": roleIDRaw,
		}).Error("Invalid role ID format in UpdateRole")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "role_id", Msg: "Invalid role ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting admin role update attempts, unique to the authenticated admin’s UUID
	rateKey := "update_role_rate:" + authUserID.String()
	// Define a constant for the maximum number of role update attempts allowed per minute by an admin
	const maxAttempts = 5
	// Retrieve the current count of role update attempts from Redis for this admin user
	attemptsCount, err := uc.Client.Get(c.Context(), rateKey).Int()
	// Check if the rate limit key doesn’t exist in Redis, indicating the first attempt in the time window
	if err == redis.Nil {
		// Set the attempt count to 0 since no previous attempts have been recorded for this admin
		attemptsCount = 0
		// Check if an unexpected Redis error occurred beyond a missing key
	} else if err != nil {
		// Log a warning to monitor Redis connectivity or configuration issues, but proceed without blocking
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserID,
		}).Warn("Failed to check role update rate limit in Redis")
	}
	// Check if the number of role update attempts exceeds the allowed maximum for this admin
	if attemptsCount >= maxAttempts {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Role update rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive attempts by this admin
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "update_role", Msg: "Too many role update attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the attempt counter in Redis to record this role update attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-minute expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Minute)

	// Define a struct to parse the JSON request body containing role update details, with validation rules
	type Request struct {
		// Define Name as an optional string field for updating the role’s name, matching model’s size constraint
		Name *string `json:"name" validate:"omitempty,min=2,max=50"`
	}

	// Allocate a new Request struct to store the parsed request body data from the admin
	req := new(Request)
	// Parse the request body strictly into the Request struct, ensuring it matches the expected JSON structure
	if err := utils.StrictBodyParser(c, req); err != nil {
		// Log an error with details to debug parsing issues with the request body submitted by the admin
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body in UpdateRole")
		// Return a 400 Bad Request response with a structured error indicating an invalid body
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "request", Msg: "Invalid request body"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Create a new validator instance to enforce validation rules defined in the Request struct
	validator := utils.NewValidator()
	// Validate the Request struct to ensure provided fields meet constraints
	validationErrors := validator.Validate(req)
	// Check if validation returned any errors, indicated by a non-nil ErrorResponse with non-empty Errors
	if validationErrors != nil && len(validationErrors.Errors) > 0 {
		// Log an error with details about validation failures to assist in debugging admin input issues
		logger.Log.WithFields(logrus.Fields{
			"error": validationErrors,
		}).Error("Validation failed in UpdateRole")
		// Return a 422 Unprocessable Entity response with the structured validation errors for admin correction
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": validationErrors.Errors,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Check if no updates were provided by the admin (only Name is updatable per the model)
	if req.Name == nil {
		// Log an info message to note that no changes were provided in the update request
		logger.Log.WithFields(logrus.Fields{
			"roleID": roleID,
		}).Info("No fields provided for update in UpdateRole")
		// Return a 400 Bad Request response indicating that at least one field must be updated
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "update", Msg: "No fields provided for update"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Fetch the role from the database using the provided role ID to verify its existence
	role, err := uc.userSystem.RoleBy("id = ?", roleID)
	// Check if fetching the role failed
	if err != nil {
		// Log an error with details to debug database retrieval issues for the admin
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": roleID,
		}).Error("Failed to fetch role for update")
		// Check if the error indicates the role doesn’t exist in the database
		if err == gorm.ErrRecordNotFound {
			// Return a 404 Not Found response with a structured error indicating the role wasn’t found
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "role_id", Msg: "Role not found"}},
				"status": fiber.StatusNotFound,
			})
		}
		// Return a 500 Internal Server Error for unexpected database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Prepare a map to store updates for the role table
	updates := make(map[string]interface{})
	// Initialize a slice to track which fields are being updated for the response
	var updatedFields []string
	// Check if Name is provided by the admin and add it to the updates map
	if req.Name != nil {
		updates["name"] = *req.Name
		// Track the Name field as updated in the response for admin visibility
		updatedFields = append(updatedFields, "name")
	}
	// Always update the "updated_at" field with the current timestamp to reflect the admin’s modification time
	updates["updated_at"] = time.Now()
	// Track the "updated_at" field as updated in the response for admin visibility
	updatedFields = append(updatedFields, "updated_at")

	// Update the role in the database with the prepared updates
	if err := uc.DB.Model(role).Updates(updates).Error; err != nil {
		// Log an error with details if the database operation to update the role fails
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": roleID,
		}).Error("Failed to update role in database")
		// Check if the error is due to a unique constraint violation (e.g., duplicate name)
		if err.Error() == "duplicate key value violates unique constraint" {
			// Return a 409 Conflict response with a structured error indicating the role name already exists
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "name", Msg: "Role name already exists"}},
				"status": fiber.StatusConflict,
			})
		}
		// Return a 500 Internal Server Error for other database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "role_update", Msg: "Failed to update role"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Generate a Redis key for caching the role data
	roleKey := "role:" + roleID.String()
	// Invalidate the role’s Redis cache to ensure it reflects the updated details
	err = uc.Client.Del(c.Context(), roleKey).Err()
	// Check if deleting the cache entry failed
	if err != nil {
		// Log a warning if cache deletion fails, but proceed since DB data is authoritative
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": roleID,
		}).Warn("Failed to delete role cache from Redis after update")
	}
	// Invalidate the "roles:all" cache to reflect the updated role in the list
	err = uc.Client.Del(c.Context(), "roles:all").Err()
	// Check if deleting the roles list cache entry failed
	if err != nil {
		// Log a warning if cache deletion fails, but proceed since DB data is authoritative
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Failed to delete roles list cache from Redis after update")
	}

	// Log a success message with details to confirm the role was updated by the admin
	logger.Log.WithFields(logrus.Fields{
		"roleID":        roleID,
		"name":          role.Name,
		"adminUserID":   authUserID,
		"updatedFields": updatedFields,
	}).Info("Role updated successfully by admin")
	// Return a 200 OK response with the updated fields and a success message for the admin
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":        "Role updated successfully",
		"status":         fiber.StatusOK,
		"updated_fields": updatedFields,
	})
}

// DeleteRole removes a role from the system, restricted to admins with confirmation
func (uc *UserController) DeleteRole(c *fiber.Ctx) error {
	// Attempt to retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is authenticated
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized role deletion attempts without authentication
		logger.Log.Warn("DeleteRole attempted without authenticated admin ID")
		// Return a 401 Unauthorized response with a structured error; PermissionAuth should catch this earlier
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth", Msg: "Admin authentication required"}},
			"status": fiber.StatusUnauthorized,
		})
	}

	// Parse the authenticated admin user ID into a UUID for rate limiting and logging
	authUserID, err := uuid.Parse(authUserIDRaw)
	// Check if parsing the authenticated admin user ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid authenticated admin ID formats from the context
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserIDRaw,
		}).Error("Invalid authenticated admin ID format in DeleteRole")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Get the target role ID from the URL parameter "role_id"
	roleIDRaw := c.Params("role_id")
	// Attempt to parse the target role ID as a UUID to validate its format
	roleID, err := uuid.Parse(roleIDRaw)
	// Check if parsing the target role ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid role ID formats provided in the request URL
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": roleIDRaw,
		}).Error("Invalid role ID format in DeleteRole")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "role_id", Msg: "Invalid role ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting admin role deletion attempts, unique to the authenticated admin’s UUID
	rateKey := "delete_role_rate:" + authUserID.String()
	// Define a constant for the maximum number of role deletion attempts allowed per hour by an admin
	const maxAttempts = 5
	// Retrieve the current count of role deletion attempts from Redis for this admin user
	attemptsCount, err := uc.Client.Get(c.Context(), rateKey).Int()
	// Check if the rate limit key doesn’t exist in Redis, indicating the first attempt in the time window
	if err == redis.Nil {
		// Set the attempt count to 0 since no previous attempts have been recorded for this admin
		attemptsCount = 0
		// Check if an unexpected Redis error occurred beyond a missing key
	} else if err != nil {
		// Log a warning to monitor Redis connectivity or configuration issues, but proceed without blocking
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserID,
		}).Warn("Failed to check role deletion rate limit in Redis")
	}
	// Check if the number of role deletion attempts exceeds the allowed maximum for this admin
	if attemptsCount >= maxAttempts {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Role deletion rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive attempts by this admin
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "delete_role", Msg: "Too many role deletion attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the attempt counter in Redis to record this role deletion attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-hour expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Hour)

	// Define a struct to parse the JSON request body containing confirmation for deletion
	type Request struct {
		// Define Confirm as a required boolean field to verify the admin’s intent to delete the role
		Confirm bool `json:"confirm" validate:"required"`
	}

	// Allocate a new Request struct to store the parsed request body data from the admin
	req := new(Request)
	// Parse the request body strictly into the Request struct, ensuring it matches the expected JSON structure
	if err := utils.StrictBodyParser(c, req); err != nil {
		// Log an error with details to debug parsing issues with the request body submitted by the admin
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body in DeleteRole")
		// Return a 400 Bad Request response with a structured error indicating an invalid body
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "request", Msg: "Invalid request body"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Create a new validator instance to enforce validation rules defined in the Request struct
	validator := utils.NewValidator()
	// Validate the Request struct to ensure Confirm is present and true
	validationErrors := validator.Validate(req)
	// Check if validation returned any errors or if Confirm is false
	if validationErrors != nil && len(validationErrors.Errors) > 0 || !req.Confirm {
		// Log an error with details about validation failures or missing confirmation
		logger.Log.WithFields(logrus.Fields{
			"error": validationErrors,
		}).Error("Validation failed in DeleteRole")
		// Return a 400 Bad Request response with structured errors or confirmation prompt
		if !req.Confirm {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "confirm", Msg: "Please confirm role deletion by setting 'confirm' to true"}},
				"status": fiber.StatusBadRequest,
			})
		}
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": validationErrors.Errors,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Fetch the role from the database using the provided role ID to verify its existence
	role, err := uc.userSystem.RoleBy("id = ?", roleID)
	// Check if fetching the role failed
	if err != nil {
		// Log an error with details to debug database retrieval issues for the admin
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": roleID,
		}).Error("Failed to fetch role for deletion")
		// Check if the error indicates the role doesn’t exist in the database
		if err == gorm.ErrRecordNotFound {
			// Return a 404 Not Found response with a structured error indicating the role wasn’t found
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "role_id", Msg: "Role not found"}},
				"status": fiber.StatusNotFound,
			})
		}
		// Return a 500 Internal Server Error for unexpected database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Attempt to delete the role from the database (assuming soft delete via Crud or GORM configuration)
	if err := uc.userSystem.Crud.Delete(&role, "id = ?", []interface{}{roleID}); err != nil {
		// Log an error with details if the database operation to delete the role fails
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": roleID,
		}).Error("Failed to delete role from database")
		// Return a 500 Internal Server Error with a structured error indicating the deletion failure
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "role_deletion", Msg: "Failed to delete role"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Generate a Redis key for caching the role data
	roleKey := "role:" + roleID.String()
	// Invalidate the role’s Redis cache to ensure it’s no longer accessible
	err = uc.Client.Del(c.Context(), roleKey).Err()
	// Check if deleting the cache entry failed
	if err != nil {
		// Log a warning if cache deletion fails, but proceed since DB data is authoritative
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": roleID,
		}).Warn("Failed to delete role cache from Redis after deletion")
	}
	// Invalidate the "roles:all" cache to reflect the deleted role in the list
	err = uc.Client.Del(c.Context(), "roles:all").Err()
	// Check if deleting the roles list cache entry failed
	if err != nil {
		// Log a warning if cache deletion fails, but proceed since DB data is authoritative
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Failed to delete roles list cache from Redis after deletion")
	}

	// Log a success message with details to confirm the role was deleted by the admin
	logger.Log.WithFields(logrus.Fields{
		"roleID":      roleID,
		"name":        role.Name,
		"adminUserID": authUserID,
	}).Info("Role deleted successfully by admin")
	// Return a 200 OK response with a clear success message for the admin
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Role deleted successfully",
		"status":  fiber.StatusOK,
	})
}
