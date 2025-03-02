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

// CreatePermission adds a new permission to the system, restricted to admins with specific permissions
func (uc *UserController) CreatePermission(c *fiber.Ctx) error {
	// Attempt to retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is authenticated
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized permission creation attempts without authentication
		logger.Log.Warn("CreatePermission attempted without authenticated admin ID")
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
		}).Error("Invalid authenticated admin ID format in CreatePermission")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting admin permission creation attempts, unique to the authenticated admin’s UUID
	rateKey := "create_permission_rate:" + authUserID.String()
	// Define a constant for the maximum number of permission creation attempts allowed per hour by an admin
	const maxAttempts = 5
	// Retrieve the current count of permission creation attempts from Redis for this admin user
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
		}).Warn("Failed to check permission creation rate limit in Redis")
	}
	// Check if the number of permission creation attempts exceeds the allowed maximum for this admin
	if attemptsCount >= maxAttempts {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Permission creation rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive attempts by this admin
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "create_permission", Msg: "Too many permission creation attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the attempt counter in Redis to record this permission creation attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-hour expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Hour)

	// Define a struct to parse the JSON request body containing permission details, with validation rules
	type Request struct {
		// Define Name as a required string field for the permission’s name, matching model’s size constraint
		Name string `json:"name" validate:"required,min=2,max=50"`
	}

	// Allocate a new Request struct to store the parsed request body data from the admin
	req := new(Request)
	// Parse the request body strictly into the Request struct, ensuring it matches the expected JSON structure
	if err := utils.StrictBodyParser(c, req); err != nil {
		// Log an error with details to debug parsing issues with the request body submitted by the admin
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body in CreatePermission")
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
		}).Error("Validation failed in CreatePermission")
		// Return a 422 Unprocessable Entity response with the structured validation errors for admin correction
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": validationErrors.Errors,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Create a new Permission instance with the provided name and a generated UUID
	permission := models.Permission{
		ID:   uuid.New(),
		Name: req.Name,
	}
	// Attempt to create the permission in the database using the Crud system
	if err := uc.userSystem.CreatePermission(&permission); err != nil {
		// Log an error with details if the database operation to create the permission fails (e.g., unique constraint violation)
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"name":  req.Name,
		}).Error("Failed to create permission")
		// Check if the error is due to a unique constraint violation (e.g., duplicate name)
		if err.Error() == "duplicate key value violates unique constraint" {
			// Return a 409 Conflict response with a structured error indicating the permission name already exists
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "name", Msg: "Permission name already exists"}},
				"status": fiber.StatusConflict,
			})
		}
		// Return a 500 Internal Server Error for other database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "permission_creation", Msg: "Failed to create permission"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Log a success message with details to confirm the permission was created by the admin
	logger.Log.WithFields(logrus.Fields{
		"permissionID": permission.ID,
		"name":         permission.Name,
		"adminUserID":  authUserID,
	}).Info("Permission created successfully by admin")
	// Return a 201 Created response with the permission ID and a success message for the admin
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":       "Permission created successfully",
		"permission_id": permission.ID,
		"status":        fiber.StatusCreated,
	})
}

// GetPermission retrieves details of a specific permission, accessible by admins
func (uc *UserController) GetPermission(c *fiber.Ctx) error {
	// Attempt to retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is authenticated
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized permission retrieval attempts without authentication
		logger.Log.Warn("GetPermission attempted without authenticated admin ID")
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
		}).Error("Invalid authenticated admin ID format in GetPermission")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Get the target permission ID from the URL parameter "permission_id"
	permissionIDRaw := c.Params("permission_id")
	// Attempt to parse the target permission ID as a UUID to validate its format
	permissionID, err := uuid.Parse(permissionIDRaw)
	// Check if parsing the target permission ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid permission ID formats provided in the request URL
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": permissionIDRaw,
		}).Error("Invalid permission ID format in GetPermission")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "permission_id", Msg: "Invalid permission ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting permission retrieval attempts, unique to the authenticated admin’s UUID
	rateKey := "get_permission_rate:" + authUserID.String()
	// Define a constant for the maximum number of permission retrieval attempts allowed per minute
	const maxAttempts = 100
	// Retrieve the current count of permission retrieval attempts from Redis for this admin user
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
		}).Warn("Failed to check permission retrieval rate limit in Redis")
	}
	// Check if the number of permission retrieval attempts exceeds the allowed maximum
	if attemptsCount >= maxAttempts {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Permission retrieval rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive attempts
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "get_permission", Msg: "Too many permission retrieval attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the attempt counter in Redis to record this permission retrieval attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-minute expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Minute)

	// Generate a Redis key for caching the permission data, using the target permission’s UUID
	permissionKey := "permission:" + permissionID.String()
	// Attempt to fetch the permission data from Redis for a fast response
	cachedPermission, err := uc.Client.Get(c.Context(), permissionKey).Result()
	// Declare a variable to hold the permission struct, which will be populated from Redis or DB
	var permission *models.Permission
	// Check if the permission data was successfully retrieved from Redis without errors
	if err == nil {
		// Allocate a new Permission struct to unmarshal the cached data into
		permission = &models.Permission{}
		// Unmarshal the JSON data fetched from Redis into the permission struct
		if err := json.Unmarshal([]byte(cachedPermission), permission); err != nil {
			// Log a warning if unmarshaling fails, indicating potential corruption in the cached data
			logger.Log.WithFields(logrus.Fields{
				"error":        err,
				"permissionID": permissionID,
			}).Warn("Failed to unmarshal cached permission from Redis in GetPermission")
			// Set permission to nil to force a database lookup if the cached data is unusable
			permission = nil
		}
	}
	// Check if the permission wasn’t found in Redis (cache miss) or if unmarshaling failed
	if err == redis.Nil || permission == nil {
		// Fetch the permission from the database using the PermissionBy function
		permission, err = uc.userSystem.PermissionBy("id = ?", permissionID)
		// Check if fetching the permission failed
		if err != nil {
			// Log an error with details to debug database retrieval issues for the admin
			logger.Log.WithFields(logrus.Fields{
				"error":        err,
				"permissionID": permissionID,
			}).Error("Failed to fetch permission")
			// Check if the error indicates the permission doesn’t exist in the database
			if err == gorm.ErrRecordNotFound {
				// Return a 404 Not Found response with a structured error indicating the permission wasn’t found
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"errors": []utils.Error{{Field: "permission_id", Msg: "Permission not found"}},
					"status": fiber.StatusNotFound,
				})
			}
			// Return a 500 Internal Server Error for unexpected database errors
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
				"status": fiber.StatusInternalServerError,
			})
		}
		// Marshal the permission data into JSON for caching
		permissionJSON, err := json.Marshal(permission)
		// Check if marshaling the permission data failed
		if err != nil {
			// Log a warning if marshaling fails, but proceed since we have the permission data
			logger.Log.WithFields(logrus.Fields{
				"error":        err,
				"permissionID": permissionID,
			}).Warn("Failed to marshal permission for Redis caching in GetPermission")
		} else {
			// Cache the permission data in Redis with a 30-minute TTL for quick future access
			if err := uc.Client.Set(c.Context(), permissionKey, permissionJSON, 30*time.Minute).Err(); err != nil {
				// Log a warning if caching fails, but proceed as DB data is authoritative
				logger.Log.WithFields(logrus.Fields{
					"error":        err,
					"permissionID": permissionID,
				}).Warn("Failed to cache permission in Redis")
			}
		}
	}

	// Log a success message with details to confirm the permission was retrieved by the admin
	logger.Log.WithFields(logrus.Fields{
		"permissionID": permission.ID,
		"name":         permission.Name,
		"adminUserID":  authUserID,
	}).Debug("Permission retrieved successfully by admin")
	// Return a 200 OK response with the permission details (ID, Name, CreatedAt, UpdatedAt) for the admin
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Permission retrieved successfully",
		"status":  fiber.StatusOK,
		"permission": fiber.Map{
			"id":         permission.ID,
			"name":       permission.Name,
			"created_at": permission.CreatedAt,
			"updated_at": permission.UpdatedAt,
		},
	})
}

// ListPermissions retrieves a list of all permissions, accessible by admins
func (uc *UserController) ListPermissions(c *fiber.Ctx) error {
	// Attempt to retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is authenticated
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized permission list retrieval attempts without authentication
		logger.Log.Warn("ListPermissions attempted without authenticated admin ID")
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
		}).Error("Invalid authenticated admin ID format in ListPermissions")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting permission list retrieval attempts, unique to the admin’s UUID
	rateKey := "list_permissions_rate:" + authUserID.String()
	// Define a constant for the maximum number of permission list retrieval attempts allowed per minute
	const maxAttempts = 100
	// Retrieve the current count of permission list retrieval attempts from Redis for this admin user
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
		}).Warn("Failed to check permission list retrieval rate limit in Redis")
	}
	// Check if the number of permission list retrieval attempts exceeds the allowed maximum
	if attemptsCount >= maxAttempts {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Permission list retrieval rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive attempts
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "list_permissions", Msg: "Too many permission list retrieval attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the attempt counter in Redis to record this permission list retrieval attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-minute expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Minute)

	// Generate a Redis key for caching the list of all permissions
	permissionsKey := "permissions:all"
	// Attempt to fetch the cached list of permissions from Redis for a fast response
	cachedPermissions, err := uc.Client.Get(c.Context(), permissionsKey).Result()
	// Declare a variable to hold the slice of permissions
	var permissions []models.Permission
	// Check if the permissions data was successfully retrieved from Redis without errors
	if err == nil {
		// Unmarshal the cached JSON data into the permissions slice
		if err := json.Unmarshal([]byte(cachedPermissions), &permissions); err != nil {
			// Log a warning if unmarshaling fails, indicating potential corruption in the cached data
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Warn("Failed to unmarshal cached permissions from Redis in ListPermissions")
			// Set permissions to nil to force a database lookup if the cached data is unusable
			permissions = nil
		}
	}
	// Check if the permissions weren’t found in Redis (cache miss) or if unmarshaling failed
	if err == redis.Nil || permissions == nil {
		// Fetch all permissions from the database; no DeletedAt field, assume all are active or Crud filters
		if err := uc.DB.Find(&permissions).Error; err != nil {
			// Log an error with details to debug database retrieval issues for the admin
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Error("Failed to fetch permissions from database")
			// Return a 500 Internal Server Error for unexpected database errors
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
				"status": fiber.StatusInternalServerError,
			})
		}
		// Marshal the permissions slice into JSON for caching
		permissionsJSON, err := json.Marshal(permissions)
		// Check if marshaling the permissions failed
		if err != nil {
			// Log a warning if marshaling fails, but proceed since we have the permissions data
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Warn("Failed to marshal permissions for Redis caching in ListPermissions")
		} else {
			// Cache the permissions list in Redis with a 30-minute TTL for quick future access
			if err := uc.Client.Set(c.Context(), permissionsKey, permissionsJSON, 30*time.Minute).Err(); err != nil {
				// Log a warning if caching fails, but proceed as DB data is authoritative
				logger.Log.WithFields(logrus.Fields{
					"error": err,
				}).Warn("Failed to cache permissions in Redis")
			}
		}
	}

	// Prepare a slice to hold simplified permission data for the response
	permissionList := make([]fiber.Map, len(permissions))
	// Iterate over the permissions to populate the response data
	for i, perm := range permissions {
		// Create a map for each permission with its ID, name, created_at, and updated_at fields
		permissionList[i] = fiber.Map{
			"id":         perm.ID,
			"name":       perm.Name,
			"created_at": perm.CreatedAt,
			"updated_at": perm.UpdatedAt,
		}
	}

	// Log a success message with details to confirm the permissions were retrieved by the admin
	logger.Log.WithFields(logrus.Fields{
		"adminUserID":      authUserID,
		"permission_count": len(permissions),
	}).Debug("Permissions listed successfully by admin")
	// Return a 200 OK response with the list of permissions for the admin
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":     "Permissions retrieved successfully",
		"status":      fiber.StatusOK,
		"permissions": permissionList,
	})
}

// UpdatePermission modifies an existing permission’s name, restricted to admins
func (uc *UserController) UpdatePermission(c *fiber.Ctx) error {
	// Attempt to retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is authenticated
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized permission update attempts without authentication
		logger.Log.Warn("UpdatePermission attempted without authenticated admin ID")
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
		}).Error("Invalid authenticated admin ID format in UpdatePermission")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Get the target permission ID from the URL parameter "permission_id"
	permissionIDRaw := c.Params("permission_id")
	// Attempt to parse the target permission ID as a UUID to validate its format
	permissionID, err := uuid.Parse(permissionIDRaw)
	// Check if parsing the target permission ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid permission ID formats provided in the request URL
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": permissionIDRaw,
		}).Error("Invalid permission ID format in UpdatePermission")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "permission_id", Msg: "Invalid permission ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting admin permission update attempts, unique to the admin’s UUID
	rateKey := "update_permission_rate:" + authUserID.String()
	// Define a constant for the maximum number of permission update attempts allowed per minute by an admin
	const maxAttempts = 5
	// Retrieve the current count of permission update attempts from Redis for this admin user
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
		}).Warn("Failed to check permission update rate limit in Redis")
	}
	// Check if the number of permission update attempts exceeds the allowed maximum for this admin
	if attemptsCount >= maxAttempts {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Permission update rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive attempts by this admin
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "update_permission", Msg: "Too many permission update attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the attempt counter in Redis to record this permission update attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-minute expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Minute)

	// Define a struct to parse the JSON request body containing permission update details, with validation rules
	type Request struct {
		// Define Name as an optional string field for updating the permission’s name, matching model’s size constraint
		Name *string `json:"name" validate:"omitempty,min=2,max=50"`
	}

	// Allocate a new Request struct to store the parsed request body data from the admin
	req := new(Request)
	// Parse the request body strictly into the Request struct, ensuring it matches the expected JSON structure
	if err := utils.StrictBodyParser(c, req); err != nil {
		// Log an error with details to debug parsing issues with the request body submitted by the admin
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body in UpdatePermission")
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
		}).Error("Validation failed in UpdatePermission")
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
			"permissionID": permissionID,
		}).Info("No fields provided for update in UpdatePermission")
		// Return a 400 Bad Request response indicating that at least one field must be updated
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "update", Msg: "No fields provided for update"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Fetch the permission from the database using the provided permission ID to verify its existence
	permission, err := uc.userSystem.PermissionBy("id = ?", permissionID)
	// Check if fetching the permission failed
	if err != nil {
		// Log an error with details to debug database retrieval issues for the admin
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": permissionID,
		}).Error("Failed to fetch permission for update")
		// Check if the error indicates the permission doesn’t exist in the database
		if err == gorm.ErrRecordNotFound {
			// Return a 404 Not Found response with a structured error indicating the permission wasn’t found
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "permission_id", Msg: "Permission not found"}},
				"status": fiber.StatusNotFound,
			})
		}
		// Return a 500 Internal Server Error for unexpected database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Prepare a map to store updates for the permission table
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

	// Update the permission in the database with the prepared updates
	if err := uc.DB.Model(permission).Updates(updates).Error; err != nil {
		// Log an error with details if the database operation to update the permission fails
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": permissionID,
		}).Error("Failed to update permission in database")
		// Check if the error is due to a unique constraint violation (e.g., duplicate name)
		if err.Error() == "duplicate key value violates unique constraint" {
			// Return a 409 Conflict response with a structured error indicating the permission name already exists
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "name", Msg: "Permission name already exists"}},
				"status": fiber.StatusConflict,
			})
		}
		// Return a 500 Internal Server Error for other database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "permission_update", Msg: "Failed to update permission"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Generate a Redis key for caching the permission data
	permissionKey := "permission:" + permissionID.String()
	// Invalidate the permission’s Redis cache to ensure it reflects the updated details
	err = uc.Client.Del(c.Context(), permissionKey).Err()
	// Check if deleting the cache entry failed
	if err != nil {
		// Log a warning if cache deletion fails, but proceed since DB data is authoritative
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": permissionID,
		}).Warn("Failed to delete permission cache from Redis after update")
	}
	// Invalidate the "permissions:all" cache to reflect the updated permission in the list
	err = uc.Client.Del(c.Context(), "permissions:all").Err()
	// Check if deleting the permissions list cache entry failed
	if err != nil {
		// Log a warning if cache deletion fails, but proceed since DB data is authoritative
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Failed to delete permissions list cache from Redis after update")
	}

	// Log a success message with details to confirm the permission was updated by the admin
	logger.Log.WithFields(logrus.Fields{
		"permissionID":  permissionID,
		"name":          permission.Name,
		"adminUserID":   authUserID,
		"updatedFields": updatedFields,
	}).Info("Permission updated successfully by admin")
	// Return a 200 OK response with the updated fields and a success message for the admin
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":        "Permission updated successfully",
		"status":         fiber.StatusOK,
		"updated_fields": updatedFields,
	})
}

// DeletePermission removes a permission from the system, restricted to admins with confirmation
func (uc *UserController) DeletePermission(c *fiber.Ctx) error {
	// Attempt to retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is authenticated
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized permission deletion attempts without authentication
		logger.Log.Warn("DeletePermission attempted without authenticated admin ID")
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
		}).Error("Invalid authenticated admin ID format in DeletePermission")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Get the target permission ID from the URL parameter "permission_id"
	permissionIDRaw := c.Params("permission_id")
	// Attempt to parse the target permission ID as a UUID to validate its format
	permissionID, err := uuid.Parse(permissionIDRaw)
	// Check if parsing the target permission ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid permission ID formats provided in the request URL
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": permissionIDRaw,
		}).Error("Invalid permission ID format in DeletePermission")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "permission_id", Msg: "Invalid permission ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting admin permission deletion attempts, unique to the admin’s UUID
	rateKey := "delete_permission_rate:" + authUserID.String()
	// Define a constant for the maximum number of permission deletion attempts allowed per hour by an admin
	const maxAttempts = 5
	// Retrieve the current count of permission deletion attempts from Redis for this admin user
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
		}).Warn("Failed to check permission deletion rate limit in Redis")
	}
	// Check if the number of permission deletion attempts exceeds the allowed maximum for this admin
	if attemptsCount >= maxAttempts {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Permission deletion rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive attempts by this admin
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "delete_permission", Msg: "Too many permission deletion attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the attempt counter in Redis to record this permission deletion attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-hour expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Hour)

	// Define a struct to parse the JSON request body containing confirmation for deletion
	type Request struct {
		// Define Confirm as a required boolean field to verify the admin’s intent to delete the permission
		Confirm bool `json:"confirm" validate:"required"`
	}

	// Allocate a new Request struct to store the parsed request body data from the admin
	req := new(Request)
	// Parse the request body strictly into the Request struct, ensuring it matches the expected JSON structure
	if err := utils.StrictBodyParser(c, req); err != nil {
		// Log an error with details to debug parsing issues with the request body submitted by the admin
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body in DeletePermission")
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
		}).Error("Validation failed in DeletePermission")
		// Return a 400 Bad Request response with structured errors or confirmation prompt
		if !req.Confirm {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "confirm", Msg: "Please confirm permission deletion by setting 'confirm' to true"}},
				"status": fiber.StatusBadRequest,
			})
		}
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": validationErrors.Errors,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Fetch the permission from the database using the provided permission ID to verify its existence
	permission, err := uc.userSystem.PermissionBy("id = ?", permissionID)
	// Check if fetching the permission failed
	if err != nil {
		// Log an error with details to debug database retrieval issues for the admin
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": permissionID,
		}).Error("Failed to fetch permission for deletion")
		// Check if the error indicates the permission doesn’t exist in the database
		if err == gorm.ErrRecordNotFound {
			// Return a 404 Not Found response with a structured error indicating the permission wasn’t found
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "permission_id", Msg: "Permission not found"}},
				"status": fiber.StatusNotFound,
			})
		}
		// Return a 500 Internal Server Error for unexpected database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Attempt to delete the permission from the database (assuming soft delete via Crud or GORM configuration)
	if err := uc.userSystem.Crud.Delete(permission); err != nil {
		// Log an error with details if the database operation to delete the permission fails
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": permissionID,
		}).Error("Failed to delete permission from database")
		// Return a 500 Internal Server Error with a structured error indicating the deletion failure
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "permission_deletion", Msg: "Failed to delete permission"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Generate a Redis key for caching the permission data
	permissionKey := "permission:" + permissionID.String()
	// Invalidate the permission’s Redis cache to ensure it’s no longer accessible
	err = uc.Client.Del(c.Context(), permissionKey).Err()
	// Check if deleting the cache entry failed
	if err != nil {
		// Log a warning if cache deletion fails, but proceed since DB data is authoritative
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": permissionID,
		}).Warn("Failed to delete permission cache from Redis after deletion")
	}
	// Invalidate the "permissions:all" cache to reflect the deleted permission in the list
	err = uc.Client.Del(c.Context(), "permissions:all").Err()
	// Check if deleting the permissions list cache entry failed
	if err != nil {
		// Log a warning if cache deletion fails, but proceed since DB data is authoritative
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Failed to delete permissions list cache from Redis after deletion")
	}

	// Log a success message with details to confirm the permission was deleted by the admin
	logger.Log.WithFields(logrus.Fields{
		"permissionID": permissionID,
		"name":         permission.Name,
		"adminUserID":  authUserID,
	}).Info("Permission deleted successfully by admin")
	// Return a 200 OK response with a clear success message for the admin
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Permission deleted successfully",
		"status":  fiber.StatusOK,
	})
}

// AddPermissionToRole assigns a permission to a role, restricted to admins
func (uc *UserController) AddPermissionToRole(c *fiber.Ctx) error {
	// Attempt to retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is authenticated
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized permission assignment attempts without authentication
		logger.Log.Warn("AddPermissionToRole attempted without authenticated admin ID")
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
		}).Error("Invalid authenticated admin ID format in AddPermissionToRole")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting admin permission assignment attempts, unique to the admin’s UUID
	rateKey := "add_permission_to_role_rate:" + authUserID.String()
	// Define a constant for the maximum number of permission assignment attempts allowed per hour by an admin
	const maxAttempts = 5
	// Retrieve the current count of permission assignment attempts from Redis for this admin user
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
		}).Warn("Failed to check permission assignment rate limit in Redis")
	}
	// Check if the number of permission assignment attempts exceeds the allowed maximum for this admin
	if attemptsCount >= maxAttempts {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Permission assignment rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive attempts by this admin
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "add_permission_to_role", Msg: "Too many permission assignment attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the attempt counter in Redis to record this permission assignment attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-hour expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Hour)

	// Define a struct to parse the JSON request body containing role_id and permission_id, with validation rules
	type Request struct {
		// Define RoleID as a required UUID field for the target role
		RoleID uuid.UUID `json:"role_id" validate:"required"`
		// Define PermissionID as a required UUID field for the permission to assign
		PermissionID uuid.UUID `json:"permission_id" validate:"required"`
	}

	// Allocate a new Request struct to store the parsed request body data from the admin
	req := new(Request)
	// Parse the request body strictly into the Request struct, ensuring it matches the expected JSON structure
	if err := utils.StrictBodyParser(c, req); err != nil {
		// Log an error with details to debug parsing issues with the request body submitted by the admin
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body in AddPermissionToRole")
		// Return a 400 Bad Request response with a structured error indicating an invalid body
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "request", Msg: "Invalid request body"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Create a new validator instance to enforce validation rules defined in the Request struct
	validator := utils.NewValidator()
	// Validate the Request struct to ensure RoleID and PermissionID are present and non-nil
	validationErrors := validator.Validate(req)
	// Check if validation returned any errors, indicated by a non-nil ErrorResponse with non-empty Errors
	if validationErrors != nil && len(validationErrors.Errors) > 0 {
		// Log an error with details about validation failures to assist in debugging admin input issues
		logger.Log.WithFields(logrus.Fields{
			"error": validationErrors,
		}).Error("Validation failed in AddPermissionToRole")
		// Return a 422 Unprocessable Entity response with the structured validation errors for admin correction
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": validationErrors.Errors,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Fetch the role from the database using the provided RoleID to verify its existence
	role, err := uc.userSystem.RoleBy("id = ?", req.RoleID)
	// Check if fetching the role failed
	if err != nil {
		// Log an error with details to debug database retrieval issues for the role
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": req.RoleID,
		}).Error("Failed to fetch role for permission assignment")
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

	// Fetch the permission from the database using the provided PermissionID to verify its existence
	permission, err := uc.userSystem.PermissionBy("id = ?", req.PermissionID)
	// Check if fetching the permission failed
	if err != nil {
		// Log an error with details to debug database retrieval issues for the permission
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": req.PermissionID,
		}).Error("Failed to fetch permission for assignment")
		// Check if the error indicates the permission doesn’t exist in the database
		if err == gorm.ErrRecordNotFound {
			// Return a 404 Not Found response with a structured error indicating the permission wasn’t found
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "permission_id", Msg: "Permission not found"}},
				"status": fiber.StatusNotFound,
			})
		}
		// Return a 500 Internal Server Error for unexpected database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Add the permission to the role’s list of permissions in the many-to-many relationship table
	if err := uc.userSystem.Crud.AddManyToMany(role, "Permissions", permission); err != nil {
		// Log an error with details if the database operation to assign the permission fails
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"roleID":       req.RoleID,
			"permissionID": req.PermissionID,
		}).Error("Failed to assign permission to role")
		// Return a 500 Internal Server Error with a structured error indicating the assignment failure
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "permission_assignment", Msg: "Failed to assign permission to role"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Generate a Redis key for caching the role data
	roleKey := "role:" + req.RoleID.String()
	// Invalidate the role’s Redis cache to ensure it reflects the updated permissions
	err = uc.Client.Del(c.Context(), roleKey).Err()
	// Check if deleting the cache entry failed
	if err != nil {
		// Log a warning if cache deletion fails, but proceed since DB data is authoritative
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": req.RoleID,
		}).Warn("Failed to delete role cache from Redis after permission assignment")
	}

	// Log a success message with details to confirm the permission was assigned to the role by the admin
	logger.Log.WithFields(logrus.Fields{
		"roleID":       req.RoleID,
		"permissionID": req.PermissionID,
		"roleName":     role.Name,
		"permName":     permission.Name,
		"adminUserID":  authUserID,
	}).Info("Permission assigned to role successfully by admin")
	// Return a 200 OK response with a clear success message for the admin
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Permission assigned to role successfully",
		"status":  fiber.StatusOK,
	})
}

// RemovePermissionFromRole removes a permission from a role, restricted to admins with confirmation
func (uc *UserController) RemovePermissionFromRole(c *fiber.Ctx) error {
	// Attempt to retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is authenticated
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized permission removal attempts without authentication
		logger.Log.Warn("RemovePermissionFromRole attempted without authenticated admin ID")
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
		}).Error("Invalid authenticated admin ID format in RemovePermissionFromRole")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting admin permission removal attempts, unique to the admin’s UUID
	rateKey := "remove_permission_from_role_rate:" + authUserID.String()
	// Define a constant for the maximum number of permission removal attempts allowed per hour by an admin
	const maxAttempts = 5
	// Retrieve the current count of permission removal attempts from Redis for this admin user
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
		}).Warn("Failed to check permission removal rate limit in Redis")
	}
	// Check if the number of permission removal attempts exceeds the allowed maximum for this admin
	if attemptsCount >= maxAttempts {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Permission removal rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive attempts by this admin
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "remove_permission_from_role", Msg: "Too many permission removal attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the attempt counter in Redis to record this permission removal attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-hour expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Hour)

	// Define a struct to parse the JSON request body containing role_id, permission_id, and confirmation
	type Request struct {
		// Define RoleID as a required UUID field for the target role
		RoleID uuid.UUID `json:"role_id" validate:"required"`
		// Define PermissionID as a required UUID field for the permission to remove
		PermissionID uuid.UUID `json:"permission_id" validate:"required"`
		// Define Confirm as a required boolean field to verify the admin’s intent to remove the permission
		Confirm bool `json:"confirm" validate:"required"`
	}

	// Allocate a new Request struct to store the parsed request body data from the admin
	req := new(Request)
	// Parse the request body strictly into the Request struct, ensuring it matches the expected JSON structure
	if err := utils.StrictBodyParser(c, req); err != nil {
		// Log an error with details to debug parsing issues with the request body submitted by the admin
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body in RemovePermissionFromRole")
		// Return a 400 Bad Request response with a structured error indicating an invalid body
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "request", Msg: "Invalid request body"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Create a new validator instance to enforce validation rules defined in the Request struct
	validator := utils.NewValidator()
	// Validate the Request struct to ensure RoleID, PermissionID, and Confirm are present and valid
	validationErrors := validator.Validate(req)
	// Check if validation returned any errors or if Confirm is false
	if validationErrors != nil && len(validationErrors.Errors) > 0 || !req.Confirm {
		// Log an error with details about validation failures or missing confirmation
		logger.Log.WithFields(logrus.Fields{
			"error": validationErrors,
		}).Error("Validation failed in RemovePermissionFromRole")
		// Return a 400 Bad Request response with structured errors or confirmation prompt
		if !req.Confirm {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "confirm", Msg: "Please confirm permission removal by setting 'confirm' to true"}},
				"status": fiber.StatusBadRequest,
			})
		}
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": validationErrors.Errors,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Fetch the role from the database using the provided RoleID to verify its existence
	role, err := uc.userSystem.RoleBy("id = ?", req.RoleID)
	// Check if fetching the role failed
	if err != nil {
		// Log an error with details to debug database retrieval issues for the role
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": req.RoleID,
		}).Error("Failed to fetch role for permission removal")
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

	// Fetch the permission from the database using the provided PermissionID to verify its existence
	permission, err := uc.userSystem.PermissionBy("id = ?", req.PermissionID)
	// Check if fetching the permission failed
	if err != nil {
		// Log an error with details to debug database retrieval issues for the permission
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": req.PermissionID,
		}).Error("Failed to fetch permission for removal")
		// Check if the error indicates the permission doesn’t exist in the database
		if err == gorm.ErrRecordNotFound {
			// Return a 404 Not Found response with a structured error indicating the permission wasn’t found
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "permission_id", Msg: "Permission not found"}},
				"status": fiber.StatusNotFound,
			})
		}
		// Return a 500 Internal Server Error for unexpected database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Remove the specified permission from the role’s list of permissions in the many-to-many relationship table
	if err := uc.userSystem.Crud.DeleteManyToMany(role, "Permissions", permission); err != nil {
		// Log an error with details if the database operation to remove the permission fails
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"roleID":       req.RoleID,
			"permissionID": req.PermissionID,
		}).Error("Failed to remove permission from role")
		// Return a 500 Internal Server Error with a structured error indicating the removal failure
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "permission_removal", Msg: "Failed to remove permission from role"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Generate a Redis key for caching the role data
	roleKey := "role:" + req.RoleID.String()
	// Invalidate the role’s Redis cache to ensure it reflects the updated permissions list
	err = uc.Client.Del(c.Context(), roleKey).Err()
	// Check if deleting the role cache entry failed
	if err != nil {
		// Log a warning if cache deletion fails, but proceed since DB data is authoritative
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"roleID": req.RoleID,
		}).Warn("Failed to delete role cache from Redis after permission removal")
	}

	// Log a success message with details to confirm the permission was removed from the role by the admin
	logger.Log.WithFields(logrus.Fields{
		"roleID":       req.RoleID,
		"permissionID": req.PermissionID,
		"roleName":     role.Name,
		"permName":     permission.Name,
		"adminUserID":  authUserID,
	}).Info("Permission removed from role successfully by admin")
	// Return a 200 OK response with a clear success message for the admin
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Permission removed from role successfully",
		"status":  fiber.StatusOK,
	})
}
