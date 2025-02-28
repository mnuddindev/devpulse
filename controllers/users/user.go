package users

import (
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/auth"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Registration handles user registration, assigns the "member" role, and caches data in Redis
func (uc *UserController) Registration(c *fiber.Ctx) error {
	// Declare a variable to hold the user data from the request body
	var user models.User
	// Parse the request body into the user struct, strictly validating the structure
	if err := utils.StrictBodyParser(c, &user); err != nil {
		// Log an error with details if parsing fails
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Invalid request payload")
		// Return a 400 Bad Request response with the error message
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": err.Error(),
			"status": fiber.StatusBadRequest,
		})
	}

	// Create a new validator instance to check user data
	validator := utils.NewValidator()
	// Validate the user struct against defined rules (e.g., required fields)
	if err := validator.Validate(user); err != nil {
		// Log an error with user details if validation fails
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  user,
		}).Error("User validation failed while registering")
		// Return a 422 Unprocessable Entity response with validation errors
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Generate a one-time password (OTP) for user activation
	otp, err := utils.GenerateOTP()
	// Check if OTP generation failed
	if err != nil {
		// Log an error with details about the OTP generation failure
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"field": "OTP Generation",
		}).Error("OTP Generation failed")
		// Return a 500 Internal Server Error if OTP generation fails
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to generate OTP",
			"status": fiber.StatusInternalServerError,
		})
	}
	// Assign the generated OTP to the user struct
	user.OTP = otp

	// Fetch the pre-seeded "member" role from the database using Crud.GetByCondition
	var memberRole models.Role
	err = uc.userSystem.Crud.GetByCondition(&memberRole, "name = ?", []interface{}{"member"}, []string{}, "", 0, 0)
	// Check if fetching the role failed
	if err != nil {
		// Check if the error is because the role wasn’t found (shouldn’t happen with seeding)
		if err == gorm.ErrRecordNotFound {
			// Log a critical error since the member role should exist
			logger.Log.WithFields(logrus.Fields{
				"error": err,
			}).Error("Pre-seeded 'member' role not found in database")
			// Return a 500 Internal Server Error since this is a configuration issue
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Member role not found in database",
				"status": fiber.StatusInternalServerError,
			})
		}
		// Log an error for other database issues
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch member role")
		// Return a 500 Internal Server Error for unexpected database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Database error fetching member role",
			"status": fiber.StatusInternalServerError,
		})
	}
	// Assign the "member" role to the new user
	user.Roles = []models.Role{memberRole}
	// Create the user in the database with the assigned role
	newUser, err := uc.userSystem.CreateUser(&user)
	// Check if user creation failed
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Failed to register user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  err.Error(),
			"status": fiber.StatusInternalServerError,
		})
	}

	// Generate Redis keys for caching user data (by ID) and OTP
	userKey := "user:id:" + newUser.ID.String()
	otpKey := "user:" + newUser.ID.String() + ":otp"
	// Marshal the new user struct to JSON for caching
	userJSON, err := json.Marshal(newUser)
	// Check if marshaling failed
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"email": newUser.Email,
		}).Warn("Failed to marshal user for Redis caching")
	} else {
		// Cache the user data in Redis by ID with a 1-hour TTL
		if err := uc.Client.Set(c.Context(), userKey, userJSON, time.Hour).Err(); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error": err,
				"email": newUser.Email,
			}).Warn("Failed to cache user in Redis")
		}
	}
	// Cache the OTP in Redis with a 15-minute TTL
	if err := uc.Client.Set(c.Context(), otpKey, otp, 15*time.Minute).Err(); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": newUser.ID,
		}).Warn("Failed to cache OTP in Redis")
	}

	// Send an activation email with the OTP to the new user
	utils.SendActivationEmail(otp, newUser.Email, newUser.Username)

	// Return a 201 Created response with the new user’s details
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user": fiber.Map{
			"userid":   newUser.ID,
			"username": newUser.Username,
		},
		"message": "User registered successfully!!",
	})
}

// ActiveUser verifies and activates a user by OTP, using Redis caches fully
func (uc *UserController) ActiveUser(c *fiber.Ctx) error {
	// Define a struct to parse the JSON request body containing the OTP
	type Body struct {
		Otp int64 `json:"otp"`
	}
	// Declare a variable to hold the parsed request body
	var body Body
	// Parse the request body strictly into the Body struct
	if err := utils.StrictBodyParser(c, &body); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	// Parse the user ID from the URL parameter "userid"
	userID, err := uuid.Parse(c.Params("userid"))
	// Check if the user ID is invalid or empty (uuid.Nil)
	if err != nil || userID == uuid.Nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Invalid user ID")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "User not found",
			"status": fiber.StatusNotFound,
		})
	}

	// Check if the user ID is the zero UUID (additional validation)
	if userID.String() == "00000000-0000-0000-0000-000000000000" {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("User not found")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errors": "user not found",
			"status": fiber.StatusNotFound,
		})
	}

	// Generate Redis keys for user data and OTP
	userKey := "user:id:" + userID.String()
	otpKey := "user:" + userID.String() + ":otp"
	// Attempt to fetch the user’s OTP from Redis
	cachedOTP, err := uc.Client.Get(c.Context(), otpKey).Int64()
	// Declare a variable to hold the user struct
	var user *models.User
	// Check if the OTP is cached in Redis
	if err == nil {
		// Validate the OTP from the request against the cached OTP
		if body.Otp != cachedOTP {
			logger.Log.WithFields(logrus.Fields{
				"userID": userID,
			}).Error("OTP mismatch")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "OTP not matched",
				"status": fiber.StatusBadRequest,
			})
		}
		// Fetch the user from Redis by ID
		cachedUser, err := uc.Client.Get(c.Context(), userKey).Result()
		// Check if the user is cached in Redis
		if err == nil {
			user = &models.User{}
			if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
				logger.Log.WithFields(logrus.Fields{
					"error":  err,
					"userID": userID,
				}).Warn("Failed to unmarshal cached user from Redis")
				user = nil // Reset to trigger DB fetch
			}
		}
		// If not in Redis or unmarshal failed, fetch from DB
		if err == redis.Nil || user == nil {
			user, err = uc.userSystem.UserBy("id = ?", userID)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{
					"error":  err,
					"userID": userID,
				}).Error("Failed to fetch user by ID")
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error":  "User not found",
					"status": fiber.StatusNotFound,
				})
			}
			// Re-cache user data
			userJSON, _ := json.Marshal(user)
			uc.Client.Set(c.Context(), userKey, userJSON, time.Hour)
		}
	} else if err == redis.Nil {
		// If OTP cache misses, fetch user from DB
		user, err = uc.userSystem.UserBy("id = ?", userID)
		if err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Error("Failed to fetch user by ID")
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":  "User not found",
				"status": fiber.StatusNotFound,
			})
		}
		// Validate OTP against DB
		if body.Otp != user.OTP {
			logger.Log.WithFields(logrus.Fields{
				"userID": userID,
			}).Error("OTP mismatch")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "OTP not matched",
				"status": fiber.StatusBadRequest,
			})
		}
		// Cache OTP on miss
		uc.Client.Set(c.Context(), otpKey, user.OTP, 15*time.Minute)
	} else {
		// Log Redis error and fallback to DB
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Redis error fetching OTP")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Internal server error",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Check if the user is already activated
	if user.IsActive {
		logger.Log.WithFields(logrus.Fields{
			"userID": userID,
		}).Error("OTP expired or already verified")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "OTP expired or already verified",
			"status": fiber.StatusBadRequest,
		})
	}

	// Activate the user by updating the database
	if err := uc.userSystem.ActiveUser(userID); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Failed to activate user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to activate account",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Invalidate OTP cache after activation
	uc.Client.Del(c.Context(), otpKey)
	// Update cached user data with IsActive = true
	user.IsActive = true
	userJSON, _ := json.Marshal(user)
	uc.Client.Set(c.Context(), userKey, userJSON, time.Hour)

	// Log success
	logger.Log.WithFields(logrus.Fields{
		"userID": userID,
	}).Info("User activated successfully")
	// Return success response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  fiber.StatusOK,
		"message": "Account activated successfully",
		"data": fiber.Map{
			"user_id":       user.ID,
			"name":          user.FirstName + " " + user.LastName,
			"email":         user.Email,
			"profile_photo": user.AvatarUrl,
			"message":       "Your account has been activated. Please log in now!",
		},
	})
}

// Login authenticates users and generates JWT tokens, using Redis caches for performance
func (uc *UserController) Login(c *fiber.Ctx) error {
	// Define a struct to parse the login request body with validation rules
	type Login struct {
		Email    string `json:"email" validate:"required,email,min=5"`
		Password string `json:"password" validate:"required,min=6"`
	}
	// Declare a variable to hold the parsed login data
	var login Login
	// Parse the request body into the Login struct with strict validation
	if err := utils.StrictBodyParser(c, &login); err != nil {
		// Log an error if parsing fails
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body")
		// Return a 400 Bad Request response for invalid body
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	// Create a new validator instance for input validation
	validator := utils.NewValidator()
	// Validate the login struct against defined rules (e.g., email format, password length)
	if err := validator.Validate(login); err != nil {
		// Log an error if validation fails
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  login,
		}).Error("User validation failed while logging in")
		// Return a 422 Unprocessable Entity response with validation errors
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Generate a Redis key for tracking login attempts (e.g., "login_attempts:user@example.com")
	attemptKey := "login_attempts:" + login.Email
	// Set a maximum number of allowed login attempts
	const maxAttempts = 5
	// Check the number of login attempts in Redis
	attempts, err := uc.Client.Get(c.Context(), attemptKey).Int()
	// Handle the case where the key doesn’t exist yet (first attempt)
	if err == redis.Nil {
		attempts = 0
	} else if err != nil {
		// Log a warning if Redis fails (non-critical, proceed without rate limiting)
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"email": login.Email,
		}).Warn("Failed to check login attempts in Redis")
	}
	// Check if the user has exceeded the maximum login attempts
	if attempts >= maxAttempts {
		// Log an error for too many login attempts
		logger.Log.WithFields(logrus.Fields{
			"email": login.Email,
		}).Error("Too many login attempts")
		// Return a 429 Too Many Requests response
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  "Too many login attempts, please try again later",
			"status": fiber.StatusTooManyRequests,
		})
	}

	// Generate a Redis key for caching user data (e.g., "user:email:user@example.com")
	userKey := "user:email:" + login.Email
	// Declare a variable to hold the user struct
	var user *models.User
	// Attempt to fetch the user from Redis as a JSON string
	cachedUser, err := uc.Client.Get(c.Context(), userKey).Result()
	// Check if the user is cached in Redis
	if err == nil {
		// Unmarshal the cached JSON into a user struct
		user = &models.User{}
		if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
			// Log an error if unmarshaling fails (fallback to DB)
			logger.Log.WithFields(logrus.Fields{
				"error": err,
				"email": login.Email,
			}).Warn("Failed to unmarshal cached user from Redis")
			user = nil // Reset to nil to trigger DB fetch
		}
	}
	// Check if the user wasn’t found in Redis or unmarshaling failed
	if err == redis.Nil || user == nil {
		// Fetch the user from the database by email
		user, err = uc.userSystem.UserBy("email = ?", login.Email)
		// Check if fetching the user failed
		if err != nil {
			// Increment login attempts in Redis with a 5-minute TTL
			uc.Client.Incr(c.Context(), attemptKey)
			uc.Client.Expire(c.Context(), attemptKey, 5*time.Minute)
			// Log an error if the user isn’t found
			logger.Log.WithFields(logrus.Fields{
				"error": err,
				"email": login.Email,
			}).Error("Failed to fetch user by email")
			// Return a 404 Not Found response
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":  "User not found",
				"status": fiber.StatusNotFound,
			})
		}
		// Marshal the user to JSON for caching
		userJSON, err := json.Marshal(user)
		// Check if marshaling failed
		if err != nil {
			// Log a warning if marshaling fails (non-critical, proceed)
			logger.Log.WithFields(logrus.Fields{
				"error": err,
				"email": login.Email,
			}).Warn("Failed to marshal user for Redis caching")
		} else {
			// Cache the user in Redis with a 1-hour TTL
			if err := uc.Client.Set(c.Context(), userKey, userJSON, time.Hour).Err(); err != nil {
				// Log a warning if caching fails (non-critical, proceed)
				logger.Log.WithFields(logrus.Fields{
					"error": err,
					"email": login.Email,
				}).Warn("Failed to cache user in Redis")
			}
		}
	} else if err != nil {
		// Log an error if Redis fails for another reason (proceed with DB fallback)
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"email": login.Email,
		}).Error("Redis error fetching user")
		// Fetch from DB as a fallback
		user, err = uc.userSystem.UserBy("email = ?", login.Email)
		// Check if fetching the user failed
		if err != nil {
			// Increment login attempts in Redis with a 5-minute TTL
			uc.Client.Incr(c.Context(), attemptKey)
			uc.Client.Expire(c.Context(), attemptKey, 5*time.Minute)
			// Log an error if the user isn’t found
			logger.Log.WithFields(logrus.Fields{
				"error": err,
				"email": login.Email,
			}).Error("Failed to fetch user by email")
			// Return a 404 Not Found response
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":  "User not found",
				"status": fiber.StatusNotFound,
			})
		}
	}

	// Compare the provided password with the stored hashed password
	if err := utils.ComparePasswords(user.Password, login.Password); err != nil {
		// Increment login attempts in Redis with a 5-minute TTL
		uc.Client.Incr(c.Context(), attemptKey)
		uc.Client.Expire(c.Context(), attemptKey, 5*time.Minute)
		// Log an error for password mismatch
		logger.Log.WithFields(logrus.Fields{
			"email": login.Email,
		}).Error("Password mismatch")
		// Return a 401 Unauthorized response
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Email or password not matched",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Check if the user account is activated
	if !user.IsActive {
		// Increment login attempts in Redis with a 5-minute TTL
		uc.Client.Incr(c.Context(), attemptKey)
		uc.Client.Expire(c.Context(), attemptKey, 5*time.Minute)
		// Log an error if the user isn’t verified
		logger.Log.WithFields(logrus.Fields{
			"email": login.Email,
		}).Error("User not verified")
		// Return a 401 Unauthorized response
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Verify your account first",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Reset login attempts in Redis on successful login
	uc.Client.Del(c.Context(), attemptKey)

	// Collect role IDs for JWT generation (permissions handled by RefreshTokenMiddleware)
	var roleIDs []uuid.UUID
	for _, role := range user.Roles {
		// Append each role ID to the roleIDs slice
		roleIDs = append(roleIDs, role.ID)
	}

	// Generate JWT access and refresh tokens with role IDs
	atoken, rtoken, err := auth.GenerateJWT(*user, roleIDs)
	// Check if token generation failed
	if err != nil {
		// Log an error if JWT generation fails
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to generate JWT tokens")
		// Return a 422 Unprocessable Entity response
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  "Token generation failed",
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Set the access token cookie with secure settings
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    atoken,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Strict",
	})
	// Set the refresh token cookie with secure settings
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    rtoken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   false, // Set to true in production
		SameSite: "Strict",
	})

	// Log a success message for the login
	logger.Log.WithFields(logrus.Fields{
		"email": login.Email,
	}).Info("User logged in successfully")
	// Return a 200 OK response with user details
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login successful",
		"status":  fiber.StatusOK,
		"data": fiber.Map{
			"user_id":       user.ID,
			"name":          user.FirstName + " " + user.LastName,
			"email":         user.Email,
			"profile_photo": user.AvatarUrl,
		},
	})
}

func (uc *UserController) Logout(c *fiber.Ctx) error {
	// Invalidate access token cookie
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour), // Expire immediately
		HTTPOnly: true,
		Secure:   true,     // Add in production (HTTPS-only)
		SameSite: "Strict", // Prevent CSRF
	})

	// Invalidate refresh token cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour), // Expire immediately
		HTTPOnly: true,
		Secure:   true,     // Add in production (HTTPS-only)
		SameSite: "Strict", // Prevent CSRF
	})

	// Clear Authorization header (if used)
	c.Set("Authorization", "")

	// Security headers
	c.Set("Cache-Control", "no-store")
	c.Set("Pragma", "no-cache")

	// Log the event
	logger.Log.WithFields(logrus.Fields{
		"user_id": c.Locals("user_id"),
	}).Info("User logged out")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Logout successful",
		"status":  fiber.StatusOK,
	})
}

func (uc *UserController) UserProfile(c *fiber.Ctx) error {
	// Get user ID from context
	userId, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		logger.Log.WithFields(logrus.Fields{
			"error": "User ID missing or invalid type in context",
		}).Warn("Unauthorized access attempt")
		// Return unauthorized status
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Fetch user profile from the database
	user, err := uc.userSystem.UserBy("id = ?", userId)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Database error while fetching user profile")
		// Return internal server error status
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Internal server error",
			"status": fiber.StatusInternalServerError,
		})
	}

	if user.ID.String() == "00000000-0000-0000-0000-000000000000" {
		logger.Log.WithFields(logrus.Fields{
			"error": "User not found",
		}).Warn("Unauthorized access attempt")
		// Return unauthorized status
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "User not found",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Prepare user profile response
	profileResponse := fiber.Map{
		"id":                       user.ID,
		"username":                 user.Username,
		"email":                    user.Email,
		"first_name":               user.FirstName,
		"last_name":                user.LastName,
		"bio":                      user.Bio,
		"avatar_url":               user.AvatarUrl,
		"job_title":                user.JobTitle,
		"employer":                 user.Employer,
		"location":                 user.Location,
		"github_url":               user.GithubUrl,
		"website":                  user.Website,
		"current_learning":         user.CurrentLearning,
		"available_for":            user.AvailableFor,
		"currently_hacking_on":     user.CurrentlyHackingOn,
		"pronouns":                 user.Pronouns,
		"education":                user.Education,
		"brand_color":              user.BrandColor,
		"posts_count":              user.PostsCount,
		"comments_count":           user.CommentsCount,
		"likes_count":              user.LikesCount,
		"bookmarks_count":          user.BookmarksCount,
		"last_seen":                user.LastSeen,
		"theme_preference":         user.ThemePreference,
		"base_font":                user.BaseFont,
		"site_navbar":              user.SiteNavbar,
		"content_editor":           user.ContentEditor,
		"content_mode":             user.ContentMode,
		"created_at":               user.CreatedAt,
		"updated_at":               user.UpdatedAt,
		"skills":                   user.Skills,
		"interests":                user.Interests,
		"badges":                   user.Badges,
		"roles":                    user.Roles,
		"followers":                user.Followers,
		"following":                user.Following,
		"notifications":            user.Notifications,
		"notification_preferences": user.NotificationsPreferences,
	}

	// Return user profile response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user": profileResponse,
	})
}

func (uc *UserController) UpdateUserProfile(c *fiber.Ctx) error {
	// Get user ID from context
	userid, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		logger.Log.WithFields(logrus.Fields{
			"error": "User ID missing or invalid type in context",
		}).Warn("Unauthorized access attempt")
		// Return unauthorized status
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Parse request body into updateData struct
	updateData := new(models.UpdateUser)
	if err := utils.StrictBodyParser(c, &updateData); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userid": userid,
		}).Error("Failed to parse request body")
		// Return bad request status
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate updateData
	validator := utils.NewValidator()
	if err := validator.Validate(updateData); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userid": userid,
		}).Error("User profile update validation failed while registering")
		// Return unprocessable entity status
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Find user in the database
	user, err := uc.userSystem.UserBy("id = ?", userid)
	if err != nil {
		// Return internal server error status
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile",
		})
	}

	if user.ID.String() == "00000000-0000-0000-0000-000000000000" {
		logger.Log.WithFields(logrus.Fields{
			"error": "User not found",
		}).Warn("Unauthorized access attempt")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  fiber.StatusNotFound,
			"message": "User not found!!",
		})
	}

	// Prepare updates map with non-nil fields from updateData
	updates := make(map[string]interface{})
	if updateData.Username != nil {
		updates["username"] = updateData.Username
	}
	if updateData.Email != nil {
		updates["email"] = updateData.Email
	}
	if updateData.FirstName != nil {
		updates["first_name"] = updateData.FirstName
	}
	if updateData.LastName != nil {
		updates["last_name"] = updateData.LastName
	}
	if updateData.Bio != nil {
		updates["bio"] = updateData.Bio
	}
	if updateData.AvatarUrl != nil {
		updates["avatar_url"] = updateData.AvatarUrl
	}
	if updateData.JobTitle != nil {
		updates["job_title"] = updateData.JobTitle
	}
	if updateData.Employer != nil {
		updates["employer"] = updateData.Employer
	}
	if updateData.Location != nil {
		updates["location"] = updateData.Location
	}
	if updateData.GithubUrl != nil {
		updates["github_url"] = updateData.GithubUrl
	}
	if updateData.Website != nil {
		updates["website"] = updateData.Website
	}
	if updateData.CurrentLearning != nil {
		updates["current_learning"] = updateData.CurrentLearning
	}
	if updateData.AvailableFor != nil {
		updates["available_for"] = updateData.AvailableFor
	}
	if updateData.CurrentlyHackingOn != nil {
		updates["currently_hacking_on"] = updateData.CurrentlyHackingOn
	}
	if updateData.Pronouns != nil {
		updates["pronouns"] = updateData.Pronouns
	}
	if updateData.Education != nil {
		updates["education"] = updateData.Education
	}
	if updateData.BrandColor != nil {
		updates["brand_color"] = updateData.BrandColor
	}
	if updateData.Skills != nil {
		updates["skills"] = updateData.Skills
	}
	if updateData.Interests != nil {
		updates["interests"] = updateData.Interests
	}
	updates["updated_at"] = time.Now()

	if len(updates) > 0 {
		// Update user in the database
		if err := uc.userSystem.UpdateUser("id = ?", user.ID, updates); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error": err,
				"model": "usermodel",
			}).Error("Update failed")
		}

		if updateData.Username != nil {
			user.Username = *updateData.Username
		}
		if updateData.Email != nil {
			user.Email = *updateData.Email
		}
		if updateData.FirstName != nil {
			user.FirstName = *updateData.FirstName
		}
		if updateData.LastName != nil {
			user.LastName = *updateData.LastName
		}
		if updateData.Bio != nil {
			user.Bio = *updateData.Bio
		}
		if updateData.AvatarUrl != nil {
			user.AvatarUrl = *updateData.AvatarUrl
		}
		if updateData.JobTitle != nil {
			user.JobTitle = *updateData.JobTitle
		}
		if updateData.Employer != nil {
			user.Employer = *updateData.Employer
		}
		if updateData.Location != nil {
			user.Location = *updateData.Location
		}
		if updateData.GithubUrl != nil {
			user.GithubUrl = *updateData.GithubUrl
		}
		if updateData.Website != nil {
			user.Website = *updateData.Website
		}
		if updateData.CurrentLearning != nil {
			user.CurrentLearning = *updateData.CurrentLearning
		}
		if updateData.AvailableFor != nil {
			user.AvailableFor = *updateData.AvailableFor
		}
		if updateData.CurrentlyHackingOn != nil {
			user.CurrentlyHackingOn = *updateData.CurrentlyHackingOn
		}
		if updateData.Pronouns != nil {
			user.Pronouns = *updateData.Pronouns
		}
		if updateData.Education != nil {
			user.Education = *updateData.Education
		}
		if updateData.BrandColor != nil {
			user.BrandColor = *updateData.BrandColor
		}
		if updateData.Skills != nil {
			user.Skills = *updateData.Skills
		}
		if updateData.Interests != nil {
			user.Interests = *updateData.Interests
		}
	}

	// Prepare updated user profile response
	profileResponse := fiber.Map{
		"id":                   user.ID,
		"username":             user.Username,
		"email":                user.Email,
		"first_name":           user.FirstName,
		"last_name":            user.LastName,
		"bio":                  user.Bio,
		"avatar_url":           user.AvatarUrl,
		"job_title":            user.JobTitle,
		"employer":             user.Employer,
		"location":             user.Location,
		"github_url":           user.GithubUrl,
		"website":              user.Website,
		"current_learning":     user.CurrentLearning,
		"available_for":        user.AvailableFor,
		"currently_hacking_on": user.CurrentlyHackingOn,
		"pronouns":             user.Pronouns,
		"education":            user.Education,
		"brand_color":          user.BrandColor,
		"skills":               user.Skills,
		"interests":            user.Interests,
	}

	// Return updated user profile response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user": profileResponse,
	})
}

func (uc *UserController) UpdateUserCustomization(c *fiber.Ctx) error {
	userid, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		logger.Log.WithFields(logrus.Fields{
			"error": "User ID missing or invalid type in context",
		}).Warn("Unauthorized access attempt")
		// Return unauthorized status
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	type UpdateData struct {
		ThemePreference *string `json:"theme_preference" validator:"oneof=Light Dark"`
		BaseFont        *string `json:"base_font" validator:"oneof=sans-serif sans jetbrainsmono hind-siliguri comic-sans"`
		SiteNavbar      *string `json:"site_navbar" validator:"oneof=fixed static"`
		ContentEditor   *string `json:"content_editor" validator:"oneof=rich basic"`
		ContentMode     *int    `json:"content_mode" validator:"oneof=1 2 3 4 5"`
	}

	updateData := new(UpdateData)
	if err := utils.StrictBodyParser(c, &updateData); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userid": userid,
		}).Error("Parsing Update account body failed")
		// Return bad request status
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": fiber.StatusBadRequest,
			"error":  "Failed to parse notification body",
		})
	}

	// Validate updateData
	validator := utils.NewValidator()
	if err := validator.Validate(updateData); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userid": userid,
		}).Error("User Customization update validation failed while updating")
		// Return unprocessable entity status
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Find user in the database
	user, err := uc.userSystem.UserBy("id = ?", userid)
	if err != nil {
		// Return internal server error status
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": fiber.StatusInternalServerError,
			"error":  "Failed to update profile",
		})
	}

	if user.ID.String() == "00000000-0000-0000-0000-000000000000" {
		logger.Log.WithFields(logrus.Fields{
			"error": "User not found",
		}).Warn("Unauthorized access attempt")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  fiber.StatusNotFound,
			"message": "User not found!!",
		})
	}

	updates := map[string]interface{}{}
	if updateData.ThemePreference != nil {
		updates["theme_preference"] = *updateData.ThemePreference
	}
	if updateData.BaseFont != nil {
		updates["base_font"] = *updateData.BaseFont
	}
	if updateData.SiteNavbar != nil {
		updates["site_navbar"] = *updateData.SiteNavbar
	}
	if updateData.ContentEditor != nil {
		updates["content_editor"] = *updateData.ContentEditor
	}
	if updateData.ContentMode != nil {
		updates["content_mode"] = *updateData.ContentMode
	}

	if len(updates) > 0 {
		if err := uc.userSystem.UpdateUser("id = ?", user.ID, updates); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userid": userid,
			}).Error("Update failed")
			// Return bad request status
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Failed to update users Customization",
			})
		}

		if updateData.ThemePreference != nil {
			user.ThemePreference = *updateData.ThemePreference
		}
		if updateData.BaseFont != nil {
			user.BaseFont = *updateData.BaseFont
		}
		if updateData.SiteNavbar != nil {
			user.SiteNavbar = *updateData.SiteNavbar
		}
		if updateData.ContentEditor != nil {
			user.ContentEditor = *updateData.ContentEditor
		}
		if updateData.ContentMode != nil {
			user.ContentMode = *updateData.ContentMode
		}
	}

	uu := map[string]interface{}{
		"theme_preference": user.ThemePreference,
		"base_font":        user.BaseFont,
		"site_navbar":      user.SiteNavbar,
		"content_editor":   user.ContentEditor,
		"content_mode":     user.ContentMode,
	}

	// Return success message
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  fiber.StatusOK,
		"updates": uu,
		"message": "User's customization Updated successfully!!",
	})
}

func (uc *UserController) UpdateUserNotificationsPref(c *fiber.Ctx) error {
	userid, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		logger.Log.WithFields(logrus.Fields{
			"error": "User ID missing or invalid type in context",
		}).Warn("Unauthorized access attempt")
		// Return unauthorized status
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	type UpdateData struct {
		EmailOnLikes    *bool `json:"email_on_likes" validate:"omitempty"`
		EmailOnComments *bool `json:"email_on_comments" validate:"omitempty"`
		EmailOnMentions *bool `json:"email_on_mentions" validate:"omitempty"`
		EmailOnFollower *bool `json:"email_on_followers" validate:"omitempty"`
		EmailOnBadge    *bool `json:"email_on_badge" validate:"omitempty"`
		EmailOnUnread   *bool `json:"email_on_unread" validate:"omitempty"`
		EmailOnNewPosts *bool `json:"email_on_new_posts" validate:"omitempty"`
	}

	updateData := new(UpdateData)
	if err := utils.StrictBodyParser(c, &updateData); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"model": "usermodel",
		}).Error("Parsing Update account body failed")
		// Return bad request status
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": fiber.StatusBadRequest,
			"error":  "Failed to parse notification body",
		})
	}

	// Validate updateData
	validator := utils.NewValidator()
	if err := validator.Validate(updateData); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userid": userid,
		}).Error("User notification update validation failed while registering")
		// Return unprocessable entity status
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Find user in the database
	notificationPrefs, err := uc.userSystem.NotificationPreBy("user_id = ?", userid)
	if err != nil {
		// Return internal server error status
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": fiber.StatusInternalServerError,
			"error":  "Failed to update profile",
		})
	}

	if notificationPrefs.ID.String() == "00000000-0000-0000-0000-000000000000" {
		logger.Log.WithFields(logrus.Fields{
			"error": "Notification Preferences not found",
		}).Warn("Unauthorized access attempt")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  fiber.StatusNotFound,
			"message": "Notification Preferences not found!!",
		})
	}

	updates := map[string]interface{}{}
	if updateData.EmailOnLikes != nil {
		updates["email_on_likes"] = *updateData.EmailOnLikes
	}
	if updateData.EmailOnComments != nil {
		updates["email_on_comments"] = *updateData.EmailOnComments
	}
	if updateData.EmailOnMentions != nil {
		updates["email_on_mentions"] = *updateData.EmailOnMentions
	}
	if updateData.EmailOnFollower != nil {
		updates["email_on_followers"] = *updateData.EmailOnFollower
	}
	if updateData.EmailOnBadge != nil {
		updates["email_on_badge"] = *updateData.EmailOnBadge
	}
	if updateData.EmailOnUnread != nil {
		updates["email_on_unread"] = *updateData.EmailOnUnread
	}
	if updateData.EmailOnNewPosts != nil {
		updates["email_on_new_posts"] = *updateData.EmailOnNewPosts
	}

	if len(updates) > 0 {
		if err := uc.userSystem.UpdateNotificationPref("user_id = ?", notificationPrefs.UserID, updates); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userid": userid,
			}).Error("Update failed")
			// Return bad request status
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Failed to update users Notification preferences",
			})
		}

		if updateData.EmailOnLikes != nil {
			notificationPrefs.EmailOnLikes = *updateData.EmailOnLikes
		}
		if updateData.EmailOnComments != nil {
			notificationPrefs.EmailOnComments = *updateData.EmailOnComments
		}
		if updateData.EmailOnMentions != nil {
			notificationPrefs.EmailOnMentions = *updateData.EmailOnMentions
		}
		if updateData.EmailOnFollower != nil {
			notificationPrefs.EmailOnFollower = *updateData.EmailOnFollower
		}
		if updateData.EmailOnBadge != nil {
			notificationPrefs.EmailOnBadge = *updateData.EmailOnBadge
		}
		if updateData.EmailOnUnread != nil {
			notificationPrefs.EmailOnUnread = *updateData.EmailOnUnread
		}
		if updateData.EmailOnNewPosts != nil {
			notificationPrefs.EmailOnNewPosts = *updateData.EmailOnNewPosts
		}
	}

	// Return success message
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  fiber.StatusOK,
		"updates": notificationPrefs,
		"message": "User's notification preferences Updated successfully!!",
	})
}

func (uc *UserController) UpdateUserAccount(c *fiber.Ctx) error {
	// Get user ID from context
	userid, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		logger.Log.WithFields(logrus.Fields{
			"error": "User ID missing or invalid type in context",
		}).Warn("Unauthorized access attempt")
		// Return unauthorized status
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Define struct for update data
	type UpdateData struct {
		CurrentPassword string `json:"current_password" validate:"required,min=6"`
		Password        string `json:"password" validate:"required,min=6,eqfield=ConfirmPassword"`
		ConfirmPassword string `json:"confirm_password" validate:"required,min=6"`
	}

	// Parse request body into updateData struct
	updateData := new(UpdateData)
	if err := utils.StrictBodyParser(c, &updateData); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"model": "usermodel",
		}).Error("Parsing Update account body failed")
		// Return bad request status
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": fiber.StatusBadRequest,
			"error":  "Failed to parse account body",
		})
	}

	// Validate updateData
	validator := utils.NewValidator()
	if err := validator.Validate(updateData); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userid": userid,
		}).Error("User profile update validation failed while registering")
		// Return unprocessable entity status
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Find user in the database
	user, err := uc.userSystem.UserBy("id = ?", userid)
	if err != nil {
		// Return internal server error status
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": fiber.StatusInternalServerError,
			"error":  "Failed to update profile",
		})
	}

	if user.ID.String() == "00000000-0000-0000-0000-000000000000" {
		logger.Log.WithFields(logrus.Fields{
			"error": "User not found",
		}).Warn("Unauthorized access attempt")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  fiber.StatusNotFound,
			"message": "User not found!!",
		})
	}

	// Compare current password with stored password
	if err := utils.ComparePasswords(user.Password, updateData.CurrentPassword); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  userid,
		}).Error("Old password doesn't matched")
		// Return bad request status
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": fiber.StatusBadRequest,
			"error":  "Old Password mismatched",
		})
	}

	// Hash new password
	password, _ := utils.HashPassword(updateData.Password)
	updates := map[string]interface{}{
		"password": password,
	}

	// Update user password in the database
	if err := uc.userSystem.UpdateUser("id = ?", user.ID, updates); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"model": "usermodel",
		}).Error("Update failed")
		// Return bad request status
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to update user Password",
		})
	}

	// Return success message
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  fiber.StatusOK,
		"message": "Password Updated successfully!!",
	})
}

func (uc *UserController) DeleteUser(c *fiber.Ctx) error {
	// Get user ID from context
	userid, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		logger.Log.WithFields(logrus.Fields{
			"error": "User ID missing or invalid type in context",
		}).Warn("Unauthorized access attempt")
		// Return unauthorized status
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Find user in the database
	user, err := uc.userSystem.UserBy("id = ?", userid)
	if err != nil {
		// Return internal server error status
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile",
		})
	}

	if user.ID.String() == "00000000-0000-0000-0000-000000000000" {
		logger.Log.WithFields(logrus.Fields{
			"error": "User not found",
		}).Warn("Unauthorized access attempt")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  fiber.StatusNotFound,
			"message": "User not found!!",
		})
	}

	if err := uc.userSystem.DeleteUser(userid); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": "User deletation failed",
		}).Warn("User can't be deleted")
		// Return unauthorized status
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusInternalServerError,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  fiber.StatusOK,
		"message": "User deleted successfully!!",
	})
}
