package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/mnuddindev/devpulse/pkg/services/users"
	"github.com/sirupsen/logrus"
	gorm "gorm.io/gorm"
)

// RefreshTokenMiddleware handles JWT authentication and token refresh with optimized role-based permissions
func RefreshTokenMiddleware(uc *users.UserSystem, db *gorm.DB, redisClient *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Define a map of public routes that don’t require authentication
		publicRoutes := map[string]bool{
			"/login":                 true,
			"/register":              true,
			"/refresh":               true,
			"/logout":                true,
			"/user/:userid/activate": true,
		}
		// Check if the current path is a public route or matches the activation pattern
		if publicRoutes[c.Path()] || fiber.RoutePatternMatch(c.Path(), "/user/:userid/activate") {
			// Skip authentication and proceed to the next handler
			return c.Next()
		}

		// Retrieve the access token from the "access_token" cookie
		accessToken := c.Cookies("access_token")
		// Check if the access token is missing
		if accessToken == "" {
			// Log a debug message indicating no access token was found
			logger.Log.Debug("No access token found, attempting refresh")
			// Attempt to refresh the token using the refresh token
			return handleTokenRefresh(c, uc, db, redisClient)
		}

		// Verify the access token and extract its claims
		claims, err := VerifyToken(accessToken)
		// Check if token verification failed
		if err != nil {
			// Check if the error is due to an expired token
			if errors.Is(err, ErrExpiredToken) {
				// Log a debug message indicating the token expired
				logger.Log.Debug("Access token expired, attempting refresh")
				// Attempt to refresh the token
				return handleTokenRefresh(c, uc, db, redisClient)
			}
			// Log a warning with error details for other verification failures
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Warn("Access token invalid")
			// Return a 401 Unauthorized response for invalid tokens
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":  "Invalid access token",
				"status": fiber.StatusUnauthorized,
			})
		}

		// Set the user ID in the context from the token claims
		c.Locals("user_id", claims.UserID)
		// Check if role IDs are present in the token claims
		if len(claims.Permissions) == 0 {
			// Log a warning if no role IDs are found in the token
			logger.Log.WithFields(logrus.Fields{
				"userID": claims.UserID,
			}).Warn("No role IDs found in token, falling back to database")
			// Fallback to database if token lacks role IDs
			return fetchAndSetPermissionsFromDB(c, claims.UserID, db, redisClient)
		}

		// Use the role IDs directly from claims.RoleIDs (already []uuid.UUID)
		roleIDs := claims.Permissions

		// Try to fetch permissions from Redis based on role IDs
		var permissions []string
		allCached := true
		for _, roleID := range roleIDs {
			// Generate a Redis key for the role’s permissions
			roleKey := "role_perms:" + roleID.String()
			// Attempt to get permissions for this role from Redis
			cachedPerms, err := redisClient.Get(c.Context(), roleKey).Result()
			// Check if the permissions are not in the cache
			if err == redis.Nil {
				// Mark that we missed the cache for at least one role
				allCached = false
				break
			} else if err != nil {
				// Log a warning if Redis fails (non-critical, proceed to database)
				logger.Log.WithFields(logrus.Fields{
					"error":  err,
					"roleID": roleID,
				}).Warn("Redis error fetching role permissions")
			} else {
				// Split the cached permissions string and append to the list
				perms := strings.Split(cachedPerms, ",")
				permissions = append(permissions, perms...)
			}
		}

		// Check if all permissions were found in the cache
		if allCached {
			// Deduplicate permissions using a map
			permMap := make(map[string]bool)
			for _, p := range permissions {
				permMap[p] = true
			}
			permissions = nil
			for p := range permMap {
				permissions = append(permissions, p)
			}
			// Set permissions in the context from the cache
			c.Locals("permissions", permissions)
			// Log a debug message confirming cache hit
			logger.Log.WithFields(logrus.Fields{
				"userID": claims.UserID,
			}).Debug("Permissions loaded from Redis cache")
			// Proceed to the next handler
			return c.Next()
		}

		// Fallback to database if cache is incomplete
		return fetchAndSetPermissionsFromDB(c, claims.UserID, db, redisClient)
	}
}

// handleTokenRefresh refreshes tokens and updates permissions
func handleTokenRefresh(c *fiber.Ctx, uc *users.UserSystem, db *gorm.DB, redisClient *redis.Client) error {
	// Retrieve the refresh token from the "refresh_token" cookie
	refreshToken := c.Cookies("refresh_token")
	// Check if the refresh token is missing
	if refreshToken == "" {
		// Log a warning indicating the refresh token is missing
		logger.Log.WithFields(logrus.Fields{
			"error": "Refresh token not found in cookies",
		}).Warn("Refresh token missing")
		// Return a 401 Unauthorized response requiring login
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Refresh token missing, please log in again",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Verify the refresh token and extract its claims
	refreshClaims, err := VerifyToken(refreshToken)
	// Check if refresh token verification failed
	if err != nil {
		// Log a warning with error details for invalid or expired tokens
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Refresh token invalid or expired")
		// Return a 401 Unauthorized response requiring login
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Invalid or expired refresh token, please log in again",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Fetch the user from the database using the refresh token’s user ID
	user, err := uc.UserBy("id = ?", refreshClaims.UserID)
	// Check if the user fetch failed
	if err != nil {
		// Log a warning with user ID and error details
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": refreshClaims.UserID,
		}).Warn("User not found during refresh")
		// Return a 401 Unauthorized response if the user doesn’t exist
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "User not found",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Fetch the user’s roles and permissions from the database
	var dbUser models.User
	if err := db.Preload("Roles.Permissions").Where("id = ?", user.ID).First(&dbUser).Error; err != nil {
		// Log a warning if the database query fails
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": user.ID,
		}).Warn("Failed to fetch user permissions during refresh")
		// Return a 500 Internal Server Error if permissions can’t be fetched
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to fetch user permissions",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Extract role IDs and permissions from the user’s roles
	var roleIDs []uuid.UUID
	var permissions []string
	for _, role := range dbUser.Roles {
		roleIDs = append(roleIDs, role.ID)
		for _, perm := range role.Permissions {
			permissions = append(permissions, perm.Name)
		}
	}

	// Cache role-to-permission mappings in Redis
	for _, role := range dbUser.Roles {
		// Generate a Redis key for the role’s permissions
		roleKey := "role_perms:" + role.ID.String()
		// Convert the role’s permissions to a comma-separated string
		rolePerms := strings.Join(getRolePermissions(&role), ",")
		// Cache the role’s permissions for 5 minutes
		if err := redisClient.Set(c.Context(), roleKey, rolePerms, 5*time.Minute).Err(); err != nil {
			// Log a warning if caching fails (non-critical)
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"roleID": role.ID,
			}).Warn("Failed to cache role permissions")
		}
	}

	// Generate new tokens with role IDs embedded
	accessToken, newRefreshToken, err := GenerateJWT(*user, roleIDs)
	// Check if token generation failed
	if err != nil {
		// Log an error with user ID and error details
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": user.ID,
		}).Error("Token generation failed during refresh")
		// Return a 500 Internal Server Error for token generation failure
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to generate new tokens",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Set a new access token cookie with secure settings
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})
	// Set a new refresh token cookie with secure settings
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	// Set the user ID in the context
	c.Locals("user_id", user.ID)
	// Set permissions in the context
	c.Locals("permissions", permissions)
	// Log an info message confirming successful refresh
	logger.Log.WithFields(logrus.Fields{
		"userID": user.ID,
	}).Info("Tokens refreshed successfully")
	// Proceed to the next handler
	return c.Next()
}

// fetchAndSetPermissionsFromDB fetches permissions from the database and caches them
func fetchAndSetPermissionsFromDB(c *fiber.Ctx, userID uuid.UUID, db *gorm.DB, redisClient *redis.Client) error {
	// Fetch the user’s roles and permissions from the database
	var user models.User
	if err := db.Preload("Roles.Permissions").Where("id = ?", userID).First(&user).Error; err != nil {
		// Log a warning if the database query fails
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Warn("Failed to fetch user permissions")
		// Return a 500 Internal Server Error if the query fails
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to fetch user permissions",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Extract permissions from the user’s roles
	var permissions []string
	for _, role := range user.Roles {
		// Generate a Redis key for the role’s permissions
		roleKey := "role_perms:" + role.ID.String()
		// Convert the role’s permissions to a comma-separated string
		rolePerms := strings.Join(getRolePermissions(&role), ",")
		// Cache the role’s permissions for 5 minutes
		if err := redisClient.Set(c.Context(), roleKey, rolePerms, 5*time.Minute).Err(); err != nil {
			// Log a warning if caching fails (non-critical)
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"roleID": role.ID,
			}).Warn("Failed to cache role permissions")
		}
		// Append the role’s permissions to the list
		permissions = append(permissions, getRolePermissions(&role)...)
	}

	// Deduplicate permissions using a map
	permMap := make(map[string]bool)
	for _, p := range permissions {
		permMap[p] = true
	}
	permissions = nil
	for p := range permMap {
		permissions = append(permissions, p)
	}

	// Set permissions in the context
	c.Locals("permissions", permissions)
	// Log a debug message confirming database fetch
	logger.Log.WithFields(logrus.Fields{
		"userID": userID,
	}).Debug("Permissions loaded from database")
	// Proceed to the next handler
	return c.Next()
}

// getRolePermissions extracts permissions from a role
func getRolePermissions(role *models.Role) []string {
	// Initialize a slice to hold the role’s permissions
	var perms []string
	// Iterate over the role’s permissions
	for _, perm := range role.Permissions {
		// Append each permission name to the slice
		perms = append(perms, perm.Name)
	}
	// Return the list of permissions
	return perms
}
