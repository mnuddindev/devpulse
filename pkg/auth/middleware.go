package auth

import (
	"encoding/json"
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

// RefreshTokenMiddleware handles JWT authentication, token refresh, and blacklisting with optimized performance
func RefreshTokenMiddleware(uc *users.UserSystem, db *gorm.DB, redisClient *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Define public routes that don’t require authentication
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
		// Retrieve the refresh token from the "refresh_token" cookie (for blacklist check)
		refreshToken := c.Cookies("refresh_token")

		// Check if access token is present and not blacklisted
		if accessToken != "" {
			// Generate the Redis key for blacklisted access tokens
			accessTokenKey := "blacklist:access:" + accessToken
			// Check if the access token is blacklisted in Redis
			if redisClient.Exists(c.Context(), accessTokenKey).Val() > 0 {
				// Log a warning if the access token is blacklisted
				logger.Log.WithFields(logrus.Fields{
					"token": accessToken,
				}).Warn("Attempted use of blacklisted access token")
				// Return a 401 Unauthorized response for a blacklisted token
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":  "Access token has been invalidated",
					"status": fiber.StatusUnauthorized,
				})
			}
		}

		// Check if refresh token is present and not blacklisted (preemptive check)
		if refreshToken != "" {
			// Generate the Redis key for blacklisted refresh tokens
			refreshTokenKey := "blacklist:refresh:" + refreshToken
			// Check if the refresh token is blacklisted in Redis
			if redisClient.Exists(c.Context(), refreshTokenKey).Val() > 0 {
				// Log a warning if the refresh token is blacklisted
				logger.Log.WithFields(logrus.Fields{
					"token": refreshToken,
				}).Warn("Attempted use of blacklisted refresh token")
				// Return a 401 Unauthorized response for a blacklisted token
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":  "Refresh token has been invalidated",
					"status": fiber.StatusUnauthorized,
				})
			}
		}

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

		// Generate a Redis key for the cached user data
		userKey := "user:id:" + claims.UserID.String()
		// Declare a variable to hold the user struct
		var user *models.User
		// Attempt to fetch the user from Redis
		cachedUser, err := redisClient.Get(c.Context(), userKey).Result()
		// Check if the user is cached in Redis
		if err == nil {
			// Unmarshal the cached JSON into a user struct
			user = &models.User{}
			if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
				// Log a warning if unmarshaling fails (fallback to DB)
				logger.Log.WithFields(logrus.Fields{
					"error":  err,
					"userID": claims.UserID,
				}).Warn("Failed to unmarshal cached user from Redis")
				user = nil
			}
		}
		// Check if the user wasn’t found in Redis or unmarshaling failed
		if err == redis.Nil || user == nil {
			// Fetch the user from the database using the user ID from claims
			user, err = uc.UserBy("id = ?", claims.UserID)
			// Check if fetching the user failed
			if err != nil {
				// Log a warning if the user isn’t found
				logger.Log.WithFields(logrus.Fields{
					"error":  err,
					"userID": claims.UserID,
				}).Warn("User not found during access token validation")
				// Return a 401 Unauthorized response if the user doesn’t exist
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":  "User not found",
					"status": fiber.StatusUnauthorized,
				})
			}
			// Marshal the user to JSON for caching
			userJSON, err := json.Marshal(user)
			// Check if marshaling failed
			if err != nil {
				// Log a warning if marshaling fails (non-critical, proceed)
				logger.Log.WithFields(logrus.Fields{
					"error":  err,
					"userID": claims.UserID,
				}).Warn("Failed to marshal user for Redis caching")
			} else {
				// Cache the user in Redis with a 30-minute TTL
				if err := redisClient.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
					// Log a warning if caching fails (non-critical)
					logger.Log.WithFields(logrus.Fields{
						"error":  err,
						"userID": claims.UserID,
					}).Warn("Failed to cache user in Redis")
				}
			}
		} else if err != nil {
			// Log an error if Redis fails for another reason
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": claims.UserID,
			}).Error("Redis error fetching user")
			// Fall back to database if Redis fails
			user, err = uc.UserBy("id = ?", claims.UserID)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{
					"error":  err,
					"userID": claims.UserID,
				}).Warn("User not found during access token validation")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":  "User not found",
					"status": fiber.StatusUnauthorized,
				})
			}
		}

		// Set the user ID in the context for downstream handlers
		c.Locals("user_id", claims.UserID)
		// Extract role IDs from the token claims (fix from Permissions to RoleIDs)
		roleIDs := claims.Permissions // Corrected from claims.Permissions
		// Check if no role IDs are present in the token
		if len(roleIDs) == 0 {
			// Log a warning if no role IDs are found
			logger.Log.WithFields(logrus.Fields{
				"userID": claims.UserID,
			}).Warn("No role IDs found in token, falling back to database")
			// Fetch permissions from the database if token lacks role IDs
			return fetchAndSetPermissionsFromDB(c, claims.UserID, db, redisClient)
		}

		// Declare a slice to hold permissions
		var permissions []string
		// Flag to track if all permissions are cached
		allCached := true
		// Iterate over role IDs to fetch permissions
		for _, roleID := range roleIDs {
			// Generate a Redis key for the role’s permissions
			roleKey := "role_perms:" + roleID.String()
			// Attempt to fetch permissions from Redis
			cachedPerms, err := redisClient.Get(c.Context(), roleKey).Result()
			// Check if permissions are not cached
			if err == redis.Nil {
				// Mark that not all permissions are cached
				allCached = false
				break
			} else if err != nil {
				// Log a warning if Redis fails unexpectedly
				logger.Log.WithFields(logrus.Fields{
					"error":  err,
					"roleID": roleID,
				}).Warn("Redis error fetching role permissions")
			} else {
				// Split cached permissions and append to the list
				perms := strings.Split(cachedPerms, ",")
				permissions = append(permissions, perms...)
			}
		}

		// Check if all permissions were retrieved from cache
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
			// Set permissions in the context
			c.Locals("permissions", permissions)
			// Log a debug message confirming cache usage
			logger.Log.WithFields(logrus.Fields{
				"userID": claims.UserID,
			}).Debug("Permissions loaded from Redis cache")
			// Proceed to the next handler
			return c.Next()
		}

		// Fallback to database if any permissions are missing from cache
		return fetchAndSetPermissionsFromDB(c, claims.UserID, db, redisClient)
	}
}

// handleTokenRefresh attempts to refresh tokens with blacklisting checks
func handleTokenRefresh(c *fiber.Ctx, uc *users.UserSystem, db *gorm.DB, redisClient *redis.Client) error {
	// Retrieve the refresh token from the "refresh_token" cookie
	refreshToken := c.Cookies("refresh_token")
	// Check if the refresh token is missing
	if refreshToken == "" {
		// Log a warning for missing refresh token
		logger.Log.WithFields(logrus.Fields{
			"error": "Refresh token not found in cookies",
		}).Warn("Refresh token missing")
		// Return a 401 Unauthorized response requiring login
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Refresh token missing, please log in again",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Generate the Redis key for blacklisted refresh tokens
	refreshTokenKey := "blacklist:refresh:" + refreshToken
	// Check if the refresh token is blacklisted in Redis
	if redisClient.Exists(c.Context(), refreshTokenKey).Val() > 0 {
		// Log a warning if the refresh token is blacklisted
		logger.Log.WithFields(logrus.Fields{
			"token": refreshToken,
		}).Warn("Attempted use of blacklisted refresh token")
		// Return a 401 Unauthorized response for a blacklisted token
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Refresh token has been invalidated",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Verify the refresh token and extract its claims
	refreshClaims, err := VerifyToken(refreshToken)
	// Check if refresh token verification failed
	if err != nil {
		// Log a warning for invalid or expired refresh token
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Refresh token invalid or expired")
		// Return a 401 Unauthorized response requiring login
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Invalid or expired refresh token, please log in again",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Generate a Redis key for the cached user data
	userKey := "user:id:" + refreshClaims.UserID
	// Declare a variable to hold the user struct
	var user *users.User
	// Attempt to fetch the user from Redis
	cachedUser, err := redisClient.Get(c.Context(), userKey).Result()
	// Check if the user is cached in Redis
	if err == nil {
		// Unmarshal the cached JSON into a user struct
		user = &users.User{}
		if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
			// Log a warning if unmarshaling fails (fallback to DB)
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": refreshClaims.UserID,
			}).Warn("Failed to unmarshal cached user from Redis")
			user = nil
		}
	}
	// Check if the user wasn’t found in Redis or unmarshaling failed
	if err == redis.Nil || user == nil {
		// Fetch the user from the database using the refresh token’s user ID
		user, err = uc.UserBy("id = ?", refreshClaims.UserID)
		// Check if fetching the user failed
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
		// Marshal the user to JSON for caching
		userJSON, err := json.Marshal(user)
		// Check if marshaling failed
		if err != nil {
			// Log a warning if marshaling fails (non-critical)
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": refreshClaims.UserID,
			}).Warn("Failed to marshal user for Redis caching")
		} else {
			// Cache the user in Redis with a 30-minute TTL
			if err := redisClient.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
				// Log a warning if caching fails (non-critical)
				logger.Log.WithFields(logrus.Fields{
					"error":  err,
					"userID": refreshClaims.UserID,
				}).Warn("Failed to cache user in Redis")
			}
		}
	} else if err != nil {
		// Log an error if Redis fails for another reason
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": refreshClaims.UserID,
		}).Error("Redis error fetching user")
		// Fall back to database if Redis fails
		user, err = uc.UserBy("id = ?", refreshClaims.UserID)
		if err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": refreshClaims.UserID,
			}).Warn("User not found during refresh")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":  "User not found",
				"status": fiber.StatusUnauthorized,
			})
		}
	}

	// Extract role IDs and permissions from the user’s roles
	var roleIDs []uuid.UUID
	var permissions []string
	for _, role := range user.Roles {
		// Append role ID to the list
		roleIDs = append(roleIDs, role.ID)
		// Generate a Redis key for the role’s permissions
		roleKey := "role_perms:" + role.ID.String()
		// Attempt to fetch permissions from Redis
		cachedPerms, err := redisClient.Get(c.Context(), roleKey).Result()
		// Check if permissions are cached
		if err == nil {
			// Split cached permissions and append to the list
			perms := strings.Split(cachedPerms, ",")
			permissions = append(permissions, perms...)
		} else {
			// Fetch permissions from DB if not cached
			var dbRole models.Role
			if err := db.Preload("Permissions").Where("id = ?", role.ID).First(&dbRole).Error; err == nil {
				rolePerms := getRolePermissions(&dbRole)
				permissions = append(permissions, rolePerms...)
				// Cache the permissions for future use
				redisClient.Set(c.Context(), roleKey, strings.Join(rolePerms, ","), 5*time.Minute)
			}
		}
	}

	// Generate new access and refresh tokens with role IDs
	accessToken, newRefreshToken, err := GenerateJWT(*user, roleIDs)
	// Check if token generation failed
	if err != nil {
		// Log an error if token generation fails
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": user.ID,
		}).Error("Token generation failed during refresh")
		// Return a 500 Internal Server Error response
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to generate new tokens",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Set new access token cookie with secure settings
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
		Secure:   true, // Enforce HTTPS in production
		SameSite: "Strict",
	})
	// Set new refresh token cookie with secure settings
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   true, // Enforce HTTPS in production
		SameSite: "Strict",
	})

	// Set user ID and permissions in context
	c.Locals("user_id", user.ID.String())
	c.Locals("permissions", permissions)
	// Log successful refresh
	logger.Log.WithFields(logrus.Fields{
		"userID": user.ID,
	}).Info("Tokens refreshed successfully")
	// Proceed to the next handler
	return c.Next()
}

// fetchAndSetPermissionsFromDB fetches permissions from the database and caches them
func fetchAndSetPermissionsFromDB(c *fiber.Ctx, uid uuid.UUID, db *gorm.DB, redisClient *redis.Client) error {
	// Fetch the user’s roles and permissions from the database
	var user models.User
	if err := db.Preload("Roles.Permissions").Where("id = ?", uid).First(&user).Error; err != nil {
		// Log a warning if the database query fails
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": uid,
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
		"userID": uid,
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
