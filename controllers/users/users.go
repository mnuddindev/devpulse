package users

import (
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/mnuddindev/devpulse/pkg/services/users"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UserController struct {
	DB         *gorm.DB
	Client     *redis.Client
	userSystem *users.UserSystem
}

func NewUserController(userSystem *users.UserSystem, db *gorm.DB, client *redis.Client) *UserController {
	return &UserController{
		DB:         db,
		Client:     client,
		userSystem: userSystem,
	}
}

// UserByID retrieves a user’s profile by ID with Redis optimization, handling soft-deleted users
func (uc *UserController) UserByID(c *fiber.Ctx) error {
	// Attempt to parse the user ID from the URL parameter "userid" into a UUID
	userID, err := uuid.Parse(c.Params("userid"))
	// Check if parsing failed due to an invalid UUID format in the request
	if err != nil {
		// Log an error with details to debug invalid user ID formats provided by the client
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": c.Params("userid"),
		}).Error("Invalid user ID format in UserByID")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "userid", Msg: "Invalid user ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting profile lookups, tied to the client’s IP address
	rateKey := "user_by_id_rate:" + c.IP()
	// Define the maximum number of profile lookup requests allowed per minute to prevent abuse
	const maxRequests = 100
	// Retrieve the current count of requests from Redis for this IP to enforce rate limiting
	requestsCount, err := uc.Client.Get(c.Context(), rateKey).Int()
	// Check if the rate limit key doesn’t exist in Redis, indicating the first request in the time window
	if err == redis.Nil {
		// Set the request count to 0 since no previous requests have been recorded
		requestsCount = 0
		// Check if an unexpected Redis error occurred beyond a missing key
	} else if err != nil {
		// Log a warning to monitor Redis issues, but proceed without blocking the operation
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"ip":    c.IP(),
		}).Warn("Failed to check rate limit in Redis")
	}
	// Check if the number of requests exceeds the allowed maximum
	if requestsCount >= maxRequests {
		// Log a warning to track clients hitting the rate limit
		logger.Log.WithFields(logrus.Fields{
			"ip": c.IP(),
		}).Warn("Profile lookup rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive lookups
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "request", Msg: "Too many requests, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the request counter in Redis to record this lookup attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-minute expiration on the rate limit key to reset the count after the time window
	uc.Client.Expire(c.Context(), rateKey, 1*time.Minute)

	// Generate a Redis key for the cached user profile based on the user ID
	userKey := "user:id:" + userID.String()
	// Declare a variable to hold the user struct retrieved from Redis or DB
	var user *models.User
	// Attempt to fetch the user data from Redis for a fast response
	cachedUser, err := uc.Client.Get(c.Context(), userKey).Result()
	// Check if the user data was successfully retrieved from Redis
	if err == nil {
		// Allocate a new User struct to unmarshal the cached data into
		user = &models.User{}
		// Unmarshal the JSON data from Redis into the user struct
		if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
			// Log a warning if unmarshaling fails, indicating potential cache corruption
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Warn("Failed to unmarshal cached user from Redis")
			// Set user to nil to trigger a database lookup if the cached data is invalid
			user = nil
		}
	}
	// Check if the user wasn’t found in Redis (cache miss) or unmarshaling failed
	if err == redis.Nil || user == nil {
		// Fetch the user from the database using the provided user ID
		user, err = uc.userSystem.UserBy("id = ?", userID)
		// Check if fetching the user from the database failed
		if err != nil {
			// Log an error with details to debug database fetch issues
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Error("Failed to fetch user by ID from database")
			// Check if the error indicates the user wasn’t found (not soft-deleted case)
			if err == gorm.ErrRecordNotFound {
				// Return a 404 Not Found response with a structured error
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"errors": []utils.Error{{Field: "user", Msg: "User not found"}},
					"status": fiber.StatusNotFound,
				})
			}
			// Return a 500 Internal Server Error for unexpected database issues
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
				"status": fiber.StatusInternalServerError,
			})
		}
		// Check if the user is soft-deleted by verifying if DeletedAt is set or ID is zero UUID
		if user.DeletedAt.Valid || user.ID == uuid.Nil {
			// Log an info message indicating a soft-deleted user was queried
			logger.Log.WithFields(logrus.Fields{
				"userID": userID,
			}).Info("Attempted to fetch soft-deleted user")
			// Return a 404 Not Found response since soft-deleted users should be treated as inaccessible
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "user", Msg: "User not found"}},
				"status": fiber.StatusNotFound,
			})
		}
		// Marshal the fetched user data into JSON for caching
		userJSON, err := json.Marshal(user)
		// Check if marshaling the user data failed
		if err != nil {
			// Log a warning if marshaling fails, but proceed since we have the user data
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Warn("Failed to marshal user for Redis caching")
		} else {
			// Cache the user data in Redis with a 30-minute TTL for quick future access
			if err := uc.Client.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
				// Log a warning if caching fails, but proceed as DB is the source of truth
				logger.Log.WithFields(logrus.Fields{
					"error":  err,
					"userID": userID,
				}).Warn("Failed to cache user in Redis")
			}
		}
		// If user was retrieved from Redis, check for soft-delete status
	} else if user.DeletedAt.Valid || user.ID == uuid.Nil {
		// Log an info message indicating a soft-deleted user was found in cache
		logger.Log.WithFields(logrus.Fields{
			"userID": userID,
		}).Info("Attempted to fetch soft-deleted user from cache")
		// Return a 404 Not Found response since soft-deleted users are inaccessible
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "user", Msg: "User not found"}},
			"status": fiber.StatusNotFound,
		})
	}

	// Log a success message indicating the user profile was retrieved successfully
	logger.Log.WithFields(logrus.Fields{
		"userID": userID,
	}).Debug("User profile retrieved successfully")
	// Return a 200 OK response with the user profile data, using the "public" section for non-authenticated views
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User profile retrieved successfully",
		"status":  fiber.StatusOK,
		"user":    uc.BuildProfileResponse(user, "public", nil),
	})
}

// UpdateUserByID updates a user’s profile and related data by ID with Redis optimization
func (uc *UserController) UpdateUserByID(c *fiber.Ctx) error {
	// Attempt to parse the user ID from the URL parameter "userid" into a UUID
	userID, err := uuid.Parse(c.Params("userid"))
	// Check if parsing the user ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid user ID formats from the request
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": c.Params("userid"),
		}).Error("Invalid user ID format in UpdateUserByID")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "userid", Msg: "Invalid user ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Retrieve the authenticated user ID from the context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Check if the authenticated user ID is missing or invalid, indicating lack of authentication
	if !ok || authUserIDRaw == "" {
		// Log a warning to track unauthorized update attempts without authentication
		logger.Log.Warn("UpdateUserByID attempted without authenticated user ID")
		// Return a 401 Unauthorized response with a structured error
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth", Msg: "Authentication required"}},
			"status": fiber.StatusUnauthorized,
		})
	}

	// Parse the authenticated user ID into a UUID for comparison
	authUserID, err := uuid.Parse(authUserIDRaw)
	// Check if parsing the authenticated user ID failed
	if err != nil {
		// Log an error with details to debug invalid authenticated user ID formats
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserIDRaw,
		}).Error("Invalid authenticated user ID format in UpdateUserByID")
		// Return a 400 Bad Request response with a structured error
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated user ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting update attempts specific to the authenticated user
	rateKey := "update_rate:" + authUserID.String()
	// Define the maximum number of update attempts allowed per minute to prevent abuse
	const maxUpdates = 5
	// Retrieve the current count of update attempts from Redis for this user
	updatesCount, err := uc.Client.Get(c.Context(), rateKey).Int()
	// Check if the rate limit key doesn’t exist in Redis (first update attempt)
	if err == redis.Nil {
		// Set the update count to 0 since no previous attempts have been recorded
		updatesCount = 0
		// Check if an unexpected Redis error occurred beyond a missing key
	} else if err != nil {
		// Log a warning to monitor Redis issues, but proceed without blocking the operation
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserID,
		}).Warn("Failed to check update rate limit in Redis")
	}
	// Check if the number of update attempts exceeds the allowed maximum
	if updatesCount >= maxUpdates {
		// Log a warning to track users hitting the rate limit
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Update rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive updates
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "update", Msg: "Too many update attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the update counter in Redis to record this attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-minute expiration on the rate limit key to reset the count after the time window
	uc.Client.Expire(c.Context(), rateKey, 1*time.Minute)

	// Define a struct for notification preferences within the UpdateUser struct
	type NotificationPref struct {
		EmailOnLikes    *bool `json:"email_on_likes" validate:"omitempty"`
		EmailOnComments *bool `json:"email_on_comments" validate:"omitempty"`
		EmailOnMentions *bool `json:"email_on_mentions" validate:"omitempty"`
		EmailOnFollower *bool `json:"email_on_followers" validate:"omitempty"`
		EmailOnBadge    *bool `json:"email_on_badge" validate:"omitempty"`
		EmailOnUnread   *bool `json:"email_on_unread" validate:"omitempty"`
		EmailOnNewPosts *bool `json:"email_on_new_posts" validate:"omitempty"`
	}

	// Define a struct to hold the update data with all user fields and relations
	type UpdateUser struct {
		Username                 *string             `json:"username" validate:"omitempty,min=3"`
		Email                    *string             `json:"email" validate:"omitempty,email"`
		Password                 *string             `json:"password,omitempty" validate:"omitempty,min=6"`
		FirstName                *string             `json:"first_name" validate:"omitempty,min=3"`
		LastName                 *string             `json:"last_name" validate:"omitempty,min=3"`
		Bio                      *string             `json:"bio,omitempty" validate:"omitempty,max=255"`
		AvatarUrl                *string             `json:"avatar_url,omitempty" validate:"omitempty,url"`
		JobTitle                 *string             `json:"job_title,omitempty" validate:"omitempty,max=100"`
		Employer                 *string             `json:"employer,omitempty" validate:"omitempty,max=100"`
		Location                 *string             `json:"location,omitempty" validate:"omitempty,max=100"`
		GithubUrl                *string             `json:"github_url,omitempty" validate:"omitempty,url"`
		Website                  *string             `json:"website,omitempty" validate:"omitempty,url"`
		CurrentLearning          *string             `json:"current_learning,omitempty" validate:"omitempty,max=200"`
		AvailableFor             *string             `json:"available_for,omitempty" validate:"omitempty,max=200"`
		CurrentlyHackingOn       *string             `json:"currently_hacking_on,omitempty" validate:"omitempty,max=200"`
		Pronouns                 *string             `json:"pronouns,omitempty" validate:"omitempty,max=100"`
		Education                *string             `json:"education,omitempty" validate:"omitempty,max=100"`
		BrandColor               *string             `json:"brand_color,omitempty" validate:"omitempty,max=7"`
		IsActive                 *bool               `json:"is_active"`
		IsEmailVerified          *bool               `json:"is_email_verified"`
		ThemePreference          *string             `json:"theme_preference" validate:"omitempty,oneof=Light Dark"`
		BaseFont                 *string             `json:"base_font" validate:"omitempty,oneof=sans-serif sans jetbrainsmono hind-siliguri comic-sans"`
		SiteNavbar               *string             `json:"site_navbar" validate:"omitempty,oneof=fixed static"`
		ContentEditor            *string             `json:"content_editor" validate:"omitempty,oneof=rich basic"`
		ContentMode              *int                `json:"content_mode" validate:"omitempty,oneof=1 2 3 4 5"`
		Skills                   *string             `json:"skills"`
		Interests                *string             `json:"interests"`
		Badges                   *[]models.Badge     `json:"badges"`
		Roles                    *[]models.Role      `json:"roles"`
		NotificationsPreferences *[]NotificationPref `json:"notifipre"`
	}

	// Allocate a new UpdateUser struct to parse the request body into
	updateData := new(UpdateUser)
	// Parse the request body strictly into the UpdateUser struct, ensuring correct JSON structure
	if err := utils.StrictBodyParser(c, updateData); err != nil {
		// Log an error with details to debug parsing issues with the request body
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Failed to parse request body in UpdateUserByID")
		// Return a 400 Bad Request response with a structured error
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "request", Msg: "Invalid request body"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Create a new validator instance to check the update data against defined rules
	validator := utils.NewValidator()
	// Validate the update data, ensuring all provided fields meet constraints
	validationErrors := validator.Validate(updateData)
	// Check if validation returned any errors (non-empty Errors slice indicates failure)
	if validationErrors != nil && len(validationErrors.Errors) > 0 {
		// Log an error with details about validation failures for debugging
		logger.Log.WithFields(logrus.Fields{
			"error":  validationErrors,
			"userID": userID,
		}).Error("Update validation failed in UpdateUserByID")
		// Return a 422 Unprocessable Entity response with the structured validation errors
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": validationErrors.Errors,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Generate a Redis key for the cached user profile
	userKey := "user:id:" + userID.String()
	// Attempt to fetch the user from Redis first for quick access
	cachedUser, err := uc.Client.Get(c.Context(), userKey).Result()
	// Declare a variable to hold the user struct
	var user *models.User
	// Check if the user was successfully retrieved from Redis
	if err == nil {
		// Allocate a new User struct to unmarshal the cached data into
		user = &models.User{}
		// Unmarshal the cached JSON data into the user struct
		if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
			// Log a warning if unmarshaling fails, indicating potential cache corruption
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Warn("Failed to unmarshal cached user from Redis")
			// Set user to nil to trigger a database lookup
			user = nil
		}
	}
	// Check if the user wasn’t found in Redis or unmarshaling failed
	if err == redis.Nil || user == nil {
		// Fetch the user from the database using the provided user ID
		user, err = uc.userSystem.UserBy("id = ?", userID)
		// Check if fetching the user failed
		if err != nil {
			// Log an error with details to debug database fetch issues
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Error("Failed to fetch user for update")
			// Check if the user wasn’t found in the database
			if err == gorm.ErrRecordNotFound {
				// Return a 404 Not Found response with a structured error
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"errors": []utils.Error{{Field: "user", Msg: "User not found"}},
					"status": fiber.StatusNotFound,
				})
			}
			// Return a 500 Internal Server Error for unexpected database issues
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
				"status": fiber.StatusInternalServerError,
			})
		}
	}
	// Check if the user is soft-deleted by verifying DeletedAt or zero UUID
	if user.DeletedAt.Valid || user.ID == uuid.Nil {
		// Log an info message indicating an attempt to update a soft-deleted user
		logger.Log.WithFields(logrus.Fields{
			"userID": userID,
		}).Info("Attempted to update soft-deleted user")
		// Return a 404 Not Found response since soft-deleted users cannot be updated
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "user", Msg: "User not found"}},
			"status": fiber.StatusNotFound,
		})
	}

	// Prepare updates map and track changed fields for the user table
	userUpdates := make(map[string]interface{})
	var userUpdatedFields []string
	// Check and add Username to updates if provided
	if updateData.Username != nil {
		userUpdates["username"] = *updateData.Username
		userUpdatedFields = append(userUpdatedFields, "username")
	}
	// Check and add Email to updates if provided
	if updateData.Email != nil {
		userUpdates["email"] = *updateData.Email
		userUpdatedFields = append(userUpdatedFields, "email")
	}
	// Check and add Password to updates if provided, hashing it for security
	if updateData.Password != nil {
		hashedPassword, _ := utils.HashPassword(*updateData.Password)
		userUpdates["password"] = hashedPassword
		userUpdatedFields = append(userUpdatedFields, "password")
	}
	// Check and add FirstName to updates if provided
	if updateData.FirstName != nil {
		userUpdates["first_name"] = *updateData.FirstName
		userUpdatedFields = append(userUpdatedFields, "first_name")
	}
	// Check and add LastName to updates if provided
	if updateData.LastName != nil {
		userUpdates["last_name"] = *updateData.LastName
		userUpdatedFields = append(userUpdatedFields, "last_name")
	}
	// Check and add Bio to updates if provided
	if updateData.Bio != nil {
		userUpdates["bio"] = *updateData.Bio
		userUpdatedFields = append(userUpdatedFields, "bio")
	}
	// Check and add AvatarUrl to updates if provided
	if updateData.AvatarUrl != nil {
		userUpdates["avatar_url"] = *updateData.AvatarUrl
		userUpdatedFields = append(userUpdatedFields, "avatar_url")
	}
	// Check and add JobTitle to updates if provided
	if updateData.JobTitle != nil {
		userUpdates["job_title"] = *updateData.JobTitle
		userUpdatedFields = append(userUpdatedFields, "job_title")
	}
	// Check and add Employer to updates if provided
	if updateData.Employer != nil {
		userUpdates["employer"] = *updateData.Employer
		userUpdatedFields = append(userUpdatedFields, "employer")
	}
	// Check and add Location to updates if provided
	if updateData.Location != nil {
		userUpdates["location"] = *updateData.Location
		userUpdatedFields = append(userUpdatedFields, "location")
	}
	// Check and add GithubUrl to updates if provided
	if updateData.GithubUrl != nil {
		userUpdates["github_url"] = *updateData.GithubUrl
		userUpdatedFields = append(userUpdatedFields, "github_url")
	}
	// Check and add Website to updates if provided
	if updateData.Website != nil {
		userUpdates["website"] = *updateData.Website
		userUpdatedFields = append(userUpdatedFields, "website")
	}
	// Check and add CurrentLearning to updates if provided
	if updateData.CurrentLearning != nil {
		userUpdates["current_learning"] = *updateData.CurrentLearning
		userUpdatedFields = append(userUpdatedFields, "current_learning")
	}
	// Check and add AvailableFor to updates if provided
	if updateData.AvailableFor != nil {
		userUpdates["available_for"] = *updateData.AvailableFor
		userUpdatedFields = append(userUpdatedFields, "available_for")
	}
	// Check and add CurrentlyHackingOn to updates if provided
	if updateData.CurrentlyHackingOn != nil {
		userUpdates["currently_hacking_on"] = *updateData.CurrentlyHackingOn
		userUpdatedFields = append(userUpdatedFields, "currently_hacking_on")
	}
	// Check and add Pronouns to updates if provided
	if updateData.Pronouns != nil {
		userUpdates["pronouns"] = *updateData.Pronouns
		userUpdatedFields = append(userUpdatedFields, "pronouns")
	}
	// Check and add Education to updates if provided
	if updateData.Education != nil {
		userUpdates["education"] = *updateData.Education
		userUpdatedFields = append(userUpdatedFields, "education")
	}
	// Check and add BrandColor to updates if provided
	if updateData.BrandColor != nil {
		userUpdates["brand_color"] = *updateData.BrandColor
		userUpdatedFields = append(userUpdatedFields, "brand_color")
	}
	// Check and add IsActive to updates if provided
	if updateData.IsActive != nil {
		userUpdates["is_active"] = *updateData.IsActive
		userUpdatedFields = append(userUpdatedFields, "is_active")
	}
	// Check and add IsEmailVerified to updates if provided
	if updateData.IsEmailVerified != nil {
		userUpdates["is_email_verified"] = *updateData.IsEmailVerified
		userUpdatedFields = append(userUpdatedFields, "is_email_verified")
	}
	// Check and add ThemePreference to updates if provided
	if updateData.ThemePreference != nil {
		userUpdates["theme_preference"] = *updateData.ThemePreference
		userUpdatedFields = append(userUpdatedFields, "theme_preference")
	}
	// Check and add BaseFont to updates if provided
	if updateData.BaseFont != nil {
		userUpdates["base_font"] = *updateData.BaseFont
		userUpdatedFields = append(userUpdatedFields, "base_font")
	}
	// Check and add SiteNavbar to updates if provided
	if updateData.SiteNavbar != nil {
		userUpdates["site_navbar"] = *updateData.SiteNavbar
		userUpdatedFields = append(userUpdatedFields, "site_navbar")
	}
	// Check and add ContentEditor to updates if provided
	if updateData.ContentEditor != nil {
		userUpdates["content_editor"] = *updateData.ContentEditor
		userUpdatedFields = append(userUpdatedFields, "content_editor")
	}
	// Check and add ContentMode to updates if provided
	if updateData.ContentMode != nil {
		userUpdates["content_mode"] = *updateData.ContentMode
		userUpdatedFields = append(userUpdatedFields, "content_mode")
	}
	// Check and add Skills to updates if provided
	if updateData.Skills != nil {
		userUpdates["skills"] = *updateData.Skills
		userUpdatedFields = append(userUpdatedFields, "skills")
	}
	// Check and add Interests to updates if provided
	if updateData.Interests != nil {
		userUpdates["interests"] = *updateData.Interests
		userUpdatedFields = append(userUpdatedFields, "interests")
	}
	// Always update the "updated_at" field with the current timestamp
	userUpdates["updated_at"] = time.Now()
	// Track the "updated_at" field as updated
	userUpdatedFields = append(userUpdatedFields, "updated_at")

	// Prepare updates for badges if provided
	var badgeUpdatedFields []string
	if updateData.Badges != nil && len(*updateData.Badges) > 0 {
		// Extract badge names into a slice for updating
		var newBadges []string
		for _, badge := range *updateData.Badges {
			newBadges = append(newBadges, badge.Name)
			badgeUpdatedFields = append(badgeUpdatedFields, badge.Name)
		}
		// Update the user’s badges in the database
		if err := uc.userSystem.UpdateBadge(userID, newBadges); err != nil {
			// Log an error with details if badge update fails
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Error("Failed to update badges")
			// Return a 500 Internal Server Error with a structured error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "badges", Msg: "Failed to update badges"}},
				"status": fiber.StatusInternalServerError,
			})
		}
	}

	// Prepare updates for roles if provided
	var roleUpdatedFields []string
	if updateData.Roles != nil && len(*updateData.Roles) > 0 {
		// Extract role names into a slice for updating
		var newRoles []string
		for _, role := range *updateData.Roles {
			newRoles = append(newRoles, role.Name)
			roleUpdatedFields = append(roleUpdatedFields, role.Name)
		}
		// Update the user’s roles in the database
		if err := uc.userSystem.UpdateRole(userID, newRoles); err != nil {
			// Log an error with details if role update fails
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Error("Failed to update roles")
			// Return a 500 Internal Server Error with a structured error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "roles", Msg: "Failed to update roles"}},
				"status": fiber.StatusInternalServerError,
			})
		}
	}

	// Prepare updates for notification preferences if provided
	prefUpdates := make(map[string]interface{})
	var prefUpdatedFields []string
	if updateData.NotificationsPreferences != nil && len(*updateData.NotificationsPreferences) > 0 {
		// Iterate over the provided notification preferences (assuming one record per user)
		for _, newPref := range *updateData.NotificationsPreferences {
			// Check and add EmailOnLikes to updates if provided
			if newPref.EmailOnLikes != nil {
				prefUpdates["email_on_likes"] = *newPref.EmailOnLikes
				prefUpdatedFields = append(prefUpdatedFields, "email_on_likes")
			}
			// Check and add EmailOnComments to updates if provided
			if newPref.EmailOnComments != nil {
				prefUpdates["email_on_comments"] = *newPref.EmailOnComments
				prefUpdatedFields = append(prefUpdatedFields, "email_on_comments")
			}
			// Check and add EmailOnMentions to updates if provided
			if newPref.EmailOnMentions != nil {
				prefUpdates["email_on_mentions"] = *newPref.EmailOnMentions
				prefUpdatedFields = append(prefUpdatedFields, "email_on_mentions")
			}
			// Check and add EmailOnFollower to updates if provided
			if newPref.EmailOnFollower != nil {
				prefUpdates["email_on_followers"] = *newPref.EmailOnFollower
				prefUpdatedFields = append(prefUpdatedFields, "email_on_followers")
			}
			// Check and add EmailOnBadge to updates if provided
			if newPref.EmailOnBadge != nil {
				prefUpdates["email_on_badge"] = *newPref.EmailOnBadge
				prefUpdatedFields = append(prefUpdatedFields, "email_on_badge")
			}
			// Check and add EmailOnUnread to updates if provided
			if newPref.EmailOnUnread != nil {
				prefUpdates["email_on_unread"] = *newPref.EmailOnUnread
				prefUpdatedFields = append(prefUpdatedFields, "email_on_unread")
			}
			// Check and add EmailOnNewPosts to updates if provided
			if newPref.EmailOnNewPosts != nil {
				prefUpdates["email_on_new_posts"] = *newPref.EmailOnNewPosts
				prefUpdatedFields = append(prefUpdatedFields, "email_on_new_posts")
			}
		}
	}

	// Check if there are no updates to apply across all sections
	if len(userUpdates) == 0 && len(badgeUpdatedFields) == 0 && len(roleUpdatedFields) == 0 && len(prefUpdates) == 0 {
		// Log an info message indicating no changes were provided
		logger.Log.WithFields(logrus.Fields{
			"userID": userID,
		}).Info("No fields provided for update in UpdateUserByID")
		// Fetch the current notification preferences for the response
		prefs, err := uc.userSystem.NotificationPreBy("user_id = ?", userID)
		// Handle errors fetching preferences, but allow nil if not found
		if err != nil && err != gorm.ErrRecordNotFound {
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Error("Failed to fetch notification preferences for no-op response")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
				"status": fiber.StatusInternalServerError,
			})
		}
		// Return a 200 OK response with the current user data, indicating no changes
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "No changes provided for update",
			"status":  fiber.StatusOK,
			"user":    uc.BuildProfileResponse(user, "default", prefs),
		})
	}

	// Apply user updates if any fields were provided
	var updatedUser *models.User
	if len(userUpdates) > 0 {
		// Update the user in the database and retrieve the updated record
		updatedUser, err = uc.userSystem.UpdateUser("id = ?", userID, userUpdates)
		// Check if updating the user failed
		if err != nil {
			// Log an error with details about the update failure
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Error("Failed to update user in database")
			// Return a 500 Internal Server Error with a structured error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "user", Msg: "Failed to update user profile"}},
				"status": fiber.StatusInternalServerError,
			})
		}
	} else {
		// If no user updates, use the original user for the response
		updatedUser = user
	}

	// Apply notification preference updates if any fields were provided
	var updatedPrefs *models.NotificationPrefrences
	if len(prefUpdates) > 0 {
		// Update the notification preferences and retrieve the updated record
		updatedPrefs, err = uc.userSystem.UpdateNotificationPref("user_id = ?", userID, prefUpdates)
		// Check if updating notification preferences failed
		if err != nil {
			// Log an error with details about the failure
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Error("Failed to update notification preferences in database")
			// Return a 500 Internal Server Error with a structured error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "notification_preferences", Msg: "Failed to update notification preferences"}},
				"status": fiber.StatusInternalServerError,
			})
		}
	}

	// Invalidate and update the Redis cache with the updated user profile
	if err := uc.Client.Del(c.Context(), userKey).Err(); err != nil {
		// Log a warning if cache deletion fails, but proceed since DB is updated
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Warn("Failed to delete user cache from Redis")
	}
	// Marshal the updated user data into JSON for caching
	userJSON, err := json.Marshal(updatedUser)
	// Check if marshaling failed
	if err != nil {
		// Log a warning if marshaling fails, but proceed since we have the data
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Warn("Failed to marshal updated user for Redis caching")
	} else {
		// Cache the updated user profile in Redis with a 30-minute TTL
		if err := uc.Client.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
			// Log a warning if caching fails, but proceed as DB is the source of truth
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Warn("Failed to update user cache in Redis")
		}
	}

	// Combine all updated fields for the response
	allUpdatedFields := append(userUpdatedFields, badgeUpdatedFields...)
	allUpdatedFields = append(allUpdatedFields, roleUpdatedFields...)
	allUpdatedFields = append(allUpdatedFields, prefUpdatedFields...)
	// Log a success message with the user ID and updated fields
	logger.Log.WithFields(logrus.Fields{
		"userID":         userID,
		"updated_fields": allUpdatedFields,
	}).Info("User profile updated successfully")
	// Return a 200 OK response with the updated user profile and changed fields
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":        "User profile updated successfully",
		"status":         fiber.StatusOK,
		"updated_fields": allUpdatedFields,
		"user":           uc.BuildProfileResponse(updatedUser, "default", updatedPrefs),
	})
}

// DeleteUserByID deletes a user by ID, restricted to admins with specific permissions or the "admin" role
func (uc *UserController) DeleteUserByID(c *fiber.Ctx) error {
	// Attempt to parse the target user ID from the URL parameter "userid" into a UUID format
	userID, err := uuid.Parse(c.Params("userid"))
	// Check if parsing the user ID failed due to an invalid format (e.g., not a valid UUID string)
	if err != nil {
		// Log an error with details to debug invalid user ID formats provided in the admin request URL
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": c.Params("userid"),
		}).Error("Invalid user ID format in DeleteUserByID")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "userid", Msg: "Invalid user ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Retrieve the authenticated admin user ID from the Fiber context, set by RefreshTokenMiddleware
	authUserIDRaw, ok := c.Locals("user_id").(string)
	// Verify if the authenticated admin user ID exists and is a string, ensuring the request is from an authenticated admin
	if !ok || authUserIDRaw == "" {
		// Log a warning to alert about unauthorized deletion attempts without an authenticated admin ID
		logger.Log.Warn("DeleteUserByID attempted without authenticated admin ID")
		// Return a 401 Unauthorized response with a structured error; PermissionAuth should catch this earlier
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth", Msg: "Admin authentication required"}},
			"status": fiber.StatusUnauthorized,
		})
	}

	// Parse the authenticated admin user ID into a UUID for comparison and logging
	authUserID, err := uuid.Parse(authUserIDRaw)
	// Check if parsing the authenticated admin user ID failed due to an invalid format
	if err != nil {
		// Log an error with details to debug invalid authenticated admin ID formats from the context
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserIDRaw,
		}).Error("Invalid authenticated admin ID format in DeleteUserByID")
		// Return a 400 Bad Request response with a structured error indicating the format issue
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "auth_user_id", Msg: "Invalid authenticated admin ID format"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Define a struct to capture the confirmation input from the admin request body, ensuring intentional deletion
	type ConfirmData struct {
		// Define Confirm as a required boolean field to verify the admin’s intent to delete the user
		Confirm bool `json:"confirm" validate:"required"`
	}
	// Allocate a new ConfirmData struct to parse the admin’s request body into
	confirmData := new(ConfirmData)
	// Parse the request body strictly into the ConfirmData struct, enforcing proper JSON structure
	if err := utils.StrictBodyParser(c, confirmData); err != nil {
		// Log an error with details to debug parsing issues with the confirmation body submitted by the admin
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Failed to parse confirmation body in DeleteUserByID")
		// Return a 400 Bad Request response with a structured error indicating an invalid body
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "confirm", Msg: "Confirmation required in request body"}},
			"status": fiber.StatusBadRequest,
		})
	}
	// Create a new validator instance to check the confirmation data against its validation rules
	validator := utils.NewValidator()
	// Validate the ConfirmData struct to ensure the "confirm" field is present and true for the admin’s intent
	if err := validator.Validate(confirmData); err != nil || !confirmData.Confirm {
		// Return a 400 Bad Request response if confirmation is missing or false, prompting the admin to confirm
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "confirm", Msg: "Please confirm deletion by setting 'confirm' to true"}},
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting admin delete attempts, unique to the authenticated admin’s UUID
	rateKey := "delete_rate:" + authUserID.String()
	// Define a constant for the maximum number of delete attempts allowed per hour by an admin
	const maxDeletes = 5
	// Retrieve the current count of delete attempts from Redis for this admin user
	deletesCount, err := uc.Client.Get(c.Context(), rateKey).Int()
	// Check if the rate limit key doesn’t exist in Redis, indicating the first delete attempt in the time window
	if err == redis.Nil {
		// Set the delete count to 0 since no previous attempts have been recorded for this admin
		deletesCount = 0
		// Check if an unexpected Redis error occurred beyond a missing key
	} else if err != nil {
		// Log a warning to monitor Redis connectivity or configuration issues, but proceed without blocking
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": authUserID,
		}).Warn("Failed to check delete rate limit in Redis")
	}
	// Check if the number of delete attempts exceeds the allowed maximum for this admin
	if deletesCount >= maxDeletes {
		// Log a warning to track admins hitting the rate limit, aiding in monitoring potential abuse
		logger.Log.WithFields(logrus.Fields{
			"userID": authUserID,
		}).Warn("Delete rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive deletes by this admin
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "delete", Msg: "Too many delete attempts, please try again later"}},
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the delete counter in Redis to record this attempt for rate limiting
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-hour expiration on the rate limit key to reset the count after the time window expires
	uc.Client.Expire(c.Context(), rateKey, 1*time.Hour)

	// Generate a Redis key for the cached user profile, using the target user’s UUID
	userKey := "user:id:" + userID.String()
	// Check if the user exists in Redis to optimize performance by avoiding unnecessary DB calls for admins
	exists, err := uc.Client.Exists(c.Context(), userKey).Result()
	// Handle any Redis errors that might occur during the existence check
	if err != nil {
		// Log a warning to note Redis issues, but proceed to the database for verification
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Warn("Failed to check user existence in Redis")
	}
	// If the user isn’t found in Redis (cache miss or already deleted), verify existence in the database
	if exists == 0 {
		// Fetch the user from the database using the provided user ID as the condition
		user, err := uc.userSystem.UserBy("id = ?", userID)
		// Check if fetching the user failed
		if err != nil {
			// Log an error with details to debug database retrieval issues for the admin
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Error("Failed to fetch user for deletion")
			// Check if the error indicates the user doesn’t exist in the database
			if err == gorm.ErrRecordNotFound {
				// Return a 404 Not Found response with a structured error indicating the user wasn’t found
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"errors": []utils.Error{{Field: "user", Msg: "User not found"}},
					"status": fiber.StatusNotFound,
				})
			}
			// Return a 500 Internal Server Error for unexpected database errors
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "database", Msg: "Something went wrong, please try again"}},
				"status": fiber.StatusInternalServerError,
			})
		}
		// Check if the user is already soft-deleted by verifying DeletedAt or zero UUID
		if user.DeletedAt.Valid || user.ID == uuid.Nil {
			// Log an info message to note an admin attempt to delete an already soft-deleted user
			logger.Log.WithFields(logrus.Fields{
				"userID": userID,
			}).Info("Attempted to delete already soft-deleted user")
			// Return a 404 Not Found response since the user is effectively deleted
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errors": []utils.Error{{Field: "user", Msg: "User not found"}},
				"status": fiber.StatusNotFound,
			})
		}
		// No need to cache here since we’re deleting—proceed directly to deletion
	}

	// Attempt to delete the user from the database (soft delete via GORM’s DeletedAt) as an admin action
	err = uc.userSystem.DeleteUser(userID)
	// Check if deleting the user failed
	if err != nil {
		// Log an error with details to investigate why the deletion operation failed for the admin
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Failed to delete user from database")
		// Return a 500 Internal Server Error with a structured error indicating deletion failure
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": []utils.Error{{Field: "delete", Msg: "Failed to delete user"}},
			"status": fiber.StatusInternalServerError,
		})
	}

	// Remove the user’s profile from Redis to ensure no stale data remains after admin deletion
	err = uc.Client.Del(c.Context(), userKey).Err()
	// Check if deleting the cache entry failed
	if err != nil {
		// Log a warning to note the cache cleanup issue, but proceed since DB deletion succeeded
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Warn("Failed to delete user cache from Redis")
	}

	// Log a success message with details to confirm the user was deleted by the admin
	logger.Log.WithFields(logrus.Fields{
		"userID":      userID,
		"adminUserID": authUserID,
	}).Info("User deleted successfully by admin")
	// Return a 200 OK response with a clear, user-friendly success message for the admin
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User has been deleted successfully",
		"status":  fiber.StatusOK,
	})
}
