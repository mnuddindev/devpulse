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
		if err := uc.Client.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error": err,
				"email": newUser.Email,
			}).Warn("Failed to cache user in Redis")
		}
	}
	// Cache the OTP in Redis with a 15-minute TTL
	if err := uc.Client.Set(c.Context(), otpKey, otp, 30*time.Minute).Err(); err != nil {
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
		uc.Client.Set(c.Context(), otpKey, user.OTP, 30*time.Minute)
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
	uc.Client.Set(c.Context(), userKey, userJSON, 30*time.Minute)

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
			uc.Client.Expire(c.Context(), attemptKey, 15*time.Minute)
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
			if err := uc.Client.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
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
			uc.Client.Expire(c.Context(), attemptKey, 15*time.Minute)
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
		uc.Client.Expire(c.Context(), attemptKey, 15*time.Minute)
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
		uc.Client.Expire(c.Context(), attemptKey, 15*time.Minute)
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

// Logout handles user logout, invalidates tokens using Redis blacklisting
func (uc *UserController) Logout(c *fiber.Ctx) error {
	// Retrieve the user ID from the context, set by RefreshTokenMiddleware
	userID, ok := c.Locals("user_id").(string)
	// Check if user ID is missing or invalid
	if !ok || userID == "" {
		// Log a warning if user ID isn’t found (shouldn’t happen with middleware)
		logger.Log.Warn("Logout attempted without user ID in context")
		// Return a 401 Unauthorized response since user isn’t authenticated
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Parse the user ID into a UUID for consistency
	uid, err := uuid.Parse(userID)
	// Check if parsing the user ID failed
	if err != nil {
		// Log an error with details about the invalid user ID
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Invalid user ID format during logout")
		// Return a 400 Bad Request response for invalid user ID
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	// Get the access token from the cookie
	accessToken := c.Cookies("access_token")
	// Get the refresh token from the cookie
	refreshToken := c.Cookies("refresh_token")
	// Define Redis keys for blacklisting tokens
	accessTokenKey := "blacklist:access:" + accessToken
	refreshTokenKey := "blacklist:refresh:" + refreshToken

	// Check if access token exists
	if accessToken != "" {
		// Blacklist the access token in Redis with a 15-minute TTL (matches original expiration)
		if err := uc.Client.Set(c.Context(), accessTokenKey, "invalid", 15*time.Minute).Err(); err != nil {
			// Log a warning if Redis fails to blacklist the access token (non-critical, proceed)
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": uid,
			}).Warn("Failed to blacklist access token in Redis")
		}
	}
	// Check if refresh token exists
	if refreshToken != "" {
		// Blacklist the refresh token in Redis with a 7-day TTL (matches original expiration)
		if err := uc.Client.Set(c.Context(), refreshTokenKey, "invalid", 7*24*time.Hour).Err(); err != nil {
			// Log a warning if Redis fails to blacklist the refresh token (non-critical, proceed)
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": uid,
			}).Warn("Failed to blacklist refresh token in Redis")
		}
	}
	// Invalidate access token cookie by setting it to empty with a past expiration
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour), // Expire immediately
		HTTPOnly: true,
		Secure:   true,     // Enforce HTTPS in production
		SameSite: "Strict", // Prevent CSRF attacks
	})

	// Invalidate refresh token cookie by setting it to empty with a past expiration
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour), // Expire immediately
		HTTPOnly: true,
		Secure:   true,     // Enforce HTTPS in production
		SameSite: "Strict", // Prevent CSRF attacks
	})

	// Clear the Authorization header (optional, client-managed but added for completeness)
	c.Set("Authorization", "")

	// Set security headers to prevent caching of sensitive data
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	// Set Pragma header for older browsers to prevent caching
	c.Set("Pragma", "no-cache")
	// Set X-Content-Type-Options to prevent MIME-type sniffing
	c.Set("X-Content-Type-Options", "nosniff")
	// Set X-Frame-Options to prevent clickjacking
	c.Set("X-Frame-Options", "DENY")
	// Set Content-Security-Policy to restrict resource loading (basic example)
	c.Set("Content-Security-Policy", "default-src 'self'")

	// Log the logout event with user ID
	logger.Log.WithFields(logrus.Fields{
		"user_id": uid,
	}).Info("User logged out successfully")

	// Return a 200 OK response indicating successful logout
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Logout successful",
		"status":  fiber.StatusOK,
	})
}

// UserProfile retrieves and returns the authenticated user’s profile, optimized with Redis caching
func (uc *UserController) UserProfile(c *fiber.Ctx) error {
	// Retrieve the user ID from the context, set by RefreshTokenMiddleware
	userID, ok := c.Locals("user_id").(string)
	// Check if user ID is missing or invalid
	if !ok || userID == "" {
		// Log a warning for an unauthorized access attempt
		logger.Log.Warn("UserProfile attempted without user ID in context")
		// Return a 401 Unauthorized response if user ID isn’t present
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Parse the user ID into a UUID for consistency
	uid, err := uuid.Parse(userID)
	// Check if parsing the user ID failed
	if err != nil {
		// Log an error with details about the invalid user ID
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Invalid user ID format in UserProfile")
		// Return a 400 Bad Request response for invalid user ID
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting profile requests
	rateKey := "profile_rate:" + uid.String()
	// Set maximum requests allowed per minute
	const maxRequests = 10
	// Check the number of profile requests in Redis
	requests, err := uc.Client.Get(c.Context(), rateKey).Int()
	// Handle case where rate limit key doesn’t exist (first request)
	if err == redis.Nil {
		requests = 0
	} else if err != nil {
		// Log a warning if Redis fails (non-critical, proceed without rate limiting)
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": uid,
		}).Warn("Failed to check profile rate limit in Redis")
	}
	// Check if the user has exceeded the rate limit
	if requests >= maxRequests {
		// Log a warning for rate limit exceeded
		logger.Log.WithFields(logrus.Fields{
			"userID": uid,
		}).Warn("Profile request rate limit exceeded")
		// Return a 429 Too Many Requests response
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  "Too many requests, please try again later",
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the request counter with a 1-minute TTL
	uc.Client.Incr(c.Context(), rateKey)
	uc.Client.Expire(c.Context(), rateKey, 1*time.Minute)

	// Generate a Redis key for the cached user profile
	userKey := "user:id:" + uid.String()
	// Declare a variable to hold the user struct
	var user *models.User
	// Attempt to fetch the user profile from Redis
	cachedUser, err := uc.Client.Get(c.Context(), userKey).Result()
	// Check if the user profile is cached in Redis
	if err == nil {
		// Unmarshal the cached JSON into a user struct
		user = &models.User{}
		if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
			// Log a warning if unmarshaling fails (fallback to DB)
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": uid,
			}).Warn("Failed to unmarshal cached user from Redis")
			user = nil
		}
	}
	// Check if the user wasn’t found in Redis or unmarshaling failed
	if err == redis.Nil || user == nil {
		// Fetch the user profile from the database by ID
		user, err = uc.userSystem.UserBy("id = ?", uid)
		// Check if fetching the user failed
		if err != nil {
			// Log an error with details if the user isn’t found or DB fails
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": uid,
			}).Error("Database error while fetching user profile")
			// Return a 500 Internal Server Error if DB query fails
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to fetch user profile",
				"status": fiber.StatusInternalServerError,
			})
		}
		// Marshal the user to JSON for caching
		userJSON, err := json.Marshal(user)
		// Check if marshaling failed
		if err != nil {
			// Log a warning if marshaling fails (non-critical, proceed)
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": uid,
			}).Warn("Failed to marshal user for Redis caching")
		} else {
			// Cache the user profile in Redis with a 30-minute TTL
			if err := uc.Client.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
				// Log a warning if caching fails (non-critical)
				logger.Log.WithFields(logrus.Fields{
					"error":  err,
					"userID": uid,
				}).Warn("Failed to cache user profile in Redis")
			}
		}
	} else if err != nil {
		// Log an error if Redis fails for another reason (proceed with DB fallback)
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": uid,
		}).Error("Redis error fetching user profile")
		// Fetch from DB as a fallback
		user, err = uc.userSystem.UserBy("id = ?", uid)
		if err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": uid,
			}).Error("Database error while fetching user profile")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to fetch user profile",
				"status": fiber.StatusInternalServerError,
			})
		}
	}

	// Prepare user profile response with selective fields
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

	// Log successful profile retrieval
	logger.Log.WithFields(logrus.Fields{
		"userID": uid,
	}).Info("User profile retrieved successfully")
	// Return a 200 OK response with the user profile
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Profile retrieved successfully",
		"status":  fiber.StatusOK,
		"user":    profileResponse,
	})
}

// UpdateUserProfile updates the authenticated user’s profile with Redis caching and single-query perfection
func (uc *UserController) UpdateUserProfile(c *fiber.Ctx) error {
	// Retrieve the user ID from the context, set by RefreshTokenMiddleware
	userIDRaw, ok := c.Locals("user_id").(string)
	// Check if user ID is missing or invalid
	if !ok || userIDRaw == "" {
		// Log a warning for an unauthorized access attempt
		logger.Log.Warn("UpdateUserProfile attempted without user ID in context")
		// Return a 401 Unauthorized response if user ID isn’t present
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Parse the user ID into a UUID for consistency
	userID, err := uuid.Parse(userIDRaw)
	// Check if parsing the user ID failed
	if err != nil {
		// Log an error with details about the invalid user ID
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userIDRaw,
		}).Error("Invalid user ID format in UpdateUserProfile")
		// Return a 400 Bad Request response for invalid user ID
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a Redis key for rate limiting profile updates
	rateKey := "profile_update_rate:" + userID.String()
	// Set maximum updates allowed per minute
	const maxUpdates = 5
	// Check the number of update attempts in Redis
	updatesCount, err := uc.Client.Get(c.Context(), rateKey).Int()
	// Handle case where rate limit key doesn’t exist (first update)
	if err == redis.Nil {
		updatesCount = 0
	} else if err != nil {
		// Log a warning if Redis fails (non-critical, proceed without rate limiting)
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Warn("Failed to check profile update rate limit in Redis")
	}
	// Check if the user has exceeded the rate limit
	if updatesCount >= maxUpdates {
		// Log a warning for rate limit exceeded
		logger.Log.WithFields(logrus.Fields{
			"userID": userID,
		}).Warn("Profile update rate limit exceeded")
		// Return a 429 Too Many Requests response
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  "Too many update attempts, please try again later",
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the update counter with a 1-minute TTL
	uc.Client.Incr(c.Context(), rateKey)
	uc.Client.Expire(c.Context(), rateKey, 1*time.Minute)

	// Parse the request body into an UpdateUser struct
	updateData := new(models.UpdateUser)
	if err := utils.StrictBodyParser(c, updateData); err != nil {
		// Log an error if parsing fails
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Failed to parse request body in UpdateUserProfile")
		// Return a 400 Bad Request response for invalid body
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	// Create a new validator instance for input validation
	validator := utils.NewValidator()
	// Validate the update data against defined rules
	if err := validator.Validate(updateData); err != nil {
		// Log an error if validation fails
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("User profile update validation failed")
		// Return a 422 Unprocessable Entity response with validation errors
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Prepare updates map with non-nil fields from updateData using a helper function
	updates, updatedFields := uc.PrepareUserUpdates(updateData)
	// Check if there are any fields to update
	if len(updates) == 0 {
		// Fetch current user from Redis or DB for response
		user, err := uc.FetchUser(c, userID)
		if err != nil {
			return err // Error handling delegated to fetchUser
		}
		// Log an info message if no fields were updated
		logger.Log.WithFields(logrus.Fields{
			"userID": userID,
		}).Info("No fields provided for profile update")
		// Return a 200 OK response indicating no changes were made
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "No changes provided for update",
			"status":  fiber.StatusOK,
			"user":    uc.BuildProfileResponse(user, "", nil),
		})
	}

	// Update the user in the database and get the updated user back
	updatedUser, err := uc.userSystem.UpdateUser("id = ?", userID, updates)
	// Check if the update failed
	if err != nil {
		// Log an error if the update fails
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Failed to update user profile in database")
		// Return a specific error response based on the failure
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":  "User not found",
				"status": fiber.StatusNotFound,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to update profile",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Generate a Redis key for the cached user profile
	userKey := "user:id:" + userID.String()
	// Marshal the updated user to JSON for caching
	userJSON, err := json.Marshal(updatedUser)
	// Check if marshaling failed
	if err != nil {
		// Log a warning if marshaling fails (non-critical, proceed)
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Warn("Failed to marshal updated user for Redis caching")
	} else {
		// Update the Redis cache with the new user profile (30-minute TTL)
		if err := uc.Client.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
			// Log a warning if caching fails (non-critical)
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Warn("Failed to update user profile cache in Redis")
		}
	}

	// Log successful profile update with updated fields
	logger.Log.WithFields(logrus.Fields{
		"userID":         userID,
		"updated_fields": updatedFields,
	}).Info("User profile updated successfully")
	// Return a 200 OK response with the updated profile and changed fields
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":        "Profile updated successfully",
		"status":         fiber.StatusOK,
		"updated_fields": updatedFields,
		"user":           uc.BuildProfileResponse(updatedUser, "", nil),
	})
}

// prepareUserUpdates creates a map of updates and tracks changed fields for user profile updates
func (uc *UserController) PrepareUserUpdates(updateData *models.UpdateUser) (map[string]interface{}, []string) {
	// Initialize an empty map to store the fields to be updated in the database
	updates := make(map[string]interface{})
	// Initialize a slice to track which fields are being changed
	var updatedFields []string
	// Check if Username is provided in the update data and not nil
	if updateData.Username != nil {
		// Add the Username to the updates map with its new value
		updates["username"] = *updateData.Username
		// Track that the Username field is being updated
		updatedFields = append(updatedFields, "username")
	}
	// Check if Email is provided in the update data and not nil
	if updateData.Email != nil {
		// Add the Email to the updates map with its new value
		updates["email"] = *updateData.Email
		// Track that the Email field is being updated
		updatedFields = append(updatedFields, "email")
	}
	// Check if FirstName is provided in the update data and not nil
	if updateData.FirstName != nil {
		// Add the FirstName to the updates map with its new value
		updates["first_name"] = *updateData.FirstName
		// Track that the FirstName field is being updated
		updatedFields = append(updatedFields, "first_name")
	}
	// Check if LastName is provided in the update data and not nil
	if updateData.LastName != nil {
		// Add the LastName to the updates map with its new value
		updates["last_name"] = *updateData.LastName
		// Track that the LastName field is being updated
		updatedFields = append(updatedFields, "last_name")
	}
	// Check if Bio is provided in the update data and not nil
	if updateData.Bio != nil {
		// Add the Bio to the updates map with its new value
		updates["bio"] = *updateData.Bio
		// Track that the Bio field is being updated
		updatedFields = append(updatedFields, "bio")
	}
	// Check if AvatarUrl is provided in the update data and not nil
	if updateData.AvatarUrl != nil {
		// Add the AvatarUrl to the updates map with its new value
		updates["avatar_url"] = *updateData.AvatarUrl
		// Track that the AvatarUrl field is being updated
		updatedFields = append(updatedFields, "avatar_url")
	}
	// Check if JobTitle is provided in the update data and not nil
	if updateData.JobTitle != nil {
		// Add the JobTitle to the updates map with its new value
		updates["job_title"] = *updateData.JobTitle
		// Track that the JobTitle field is being updated
		updatedFields = append(updatedFields, "job_title")
	}
	// Check if Employer is provided in the update data and not nil
	if updateData.Employer != nil {
		// Add the Employer to the updates map with its new value
		updates["employer"] = *updateData.Employer
		// Track that the Employer field is being updated
		updatedFields = append(updatedFields, "employer")
	}
	// Check if Location is provided in the update data and not nil
	if updateData.Location != nil {
		// Add the Location to the updates map with its new value
		updates["location"] = *updateData.Location
		// Track that the Location field is being updated
		updatedFields = append(updatedFields, "location")
	}
	// Check if GithubUrl is provided in the update data and not nil
	if updateData.GithubUrl != nil {
		// Add the GithubUrl to the updates map with its new value
		updates["github_url"] = *updateData.GithubUrl
		// Track that the GithubUrl field is being updated
		updatedFields = append(updatedFields, "github_url")
	}
	// Check if Website is provided in the update data and not nil
	if updateData.Website != nil {
		// Add the Website to the updates map with its new value
		updates["website"] = *updateData.Website
		// Track that the Website field is being updated
		updatedFields = append(updatedFields, "website")
	}
	// Check if CurrentLearning is provided in the update data and not nil
	if updateData.CurrentLearning != nil {
		// Add the CurrentLearning to the updates map with its new value
		updates["current_learning"] = *updateData.CurrentLearning
		// Track that the CurrentLearning field is being updated
		updatedFields = append(updatedFields, "current_learning")
	}
	// Check if AvailableFor is provided in the update data and not nil
	if updateData.AvailableFor != nil {
		// Add the AvailableFor to the updates map with its new value
		updates["available_for"] = *updateData.AvailableFor
		// Track that the AvailableFor field is being updated
		updatedFields = append(updatedFields, "available_for")
	}
	// Check if CurrentlyHackingOn is provided in the update data and not nil
	if updateData.CurrentlyHackingOn != nil {
		// Add the CurrentlyHackingOn to the updates map with its new value
		updates["currently_hacking_on"] = *updateData.CurrentlyHackingOn
		// Track that the CurrentlyHackingOn field is being updated
		updatedFields = append(updatedFields, "currently_hacking_on")
	}
	// Check if Pronouns is provided in the update data and not nil
	if updateData.Pronouns != nil {
		// Add the Pronouns to the updates map with its new value
		updates["pronouns"] = *updateData.Pronouns
		// Track that the Pronouns field is being updated
		updatedFields = append(updatedFields, "pronouns")
	}
	// Check if Education is provided in the update data and not nil
	if updateData.Education != nil {
		// Add the Education to the updates map with its new value
		updates["education"] = *updateData.Education
		// Track that the Education field is being updated
		updatedFields = append(updatedFields, "education")
	}
	// Check if BrandColor is provided in the update data and not nil
	if updateData.BrandColor != nil {
		// Add the BrandColor to the updates map with its new value
		updates["brand_color"] = *updateData.BrandColor
		// Track that the BrandColor field is being updated
		updatedFields = append(updatedFields, "brand_color")
	}
	// Check if Skills is provided in the update data and not nil
	if updateData.Skills != nil {
		// Add the Skills to the updates map with its new value
		updates["skills"] = *updateData.Skills
		// Track that the Skills field is being updated
		updatedFields = append(updatedFields, "skills")
	}
	// Check if Interests is provided in the update data and not nil
	if updateData.Interests != nil {
		// Add the Interests to the updates map with its new value
		updates["interests"] = *updateData.Interests
		// Track that the Interests field is being updated
		updatedFields = append(updatedFields, "interests")
	}
	// Always update the "updated_at" field with the current timestamp
	updates["updated_at"] = time.Now()
	// Track that the "updated_at" field is being updated
	updatedFields = append(updatedFields, "updated_at")
	// Return the map of updates and the list of updated fields
	return updates, updatedFields
}

// updateUserProfileHelper updates a specific section of the user profile with Redis caching
func (uc *UserController) UpdateUserProfileHelper(c *fiber.Ctx, section string, updateData interface{}, validateFunc func(interface{}) *utils.ErrorResponse, updateFields func(*models.User, interface{}, map[string]interface{}) []string) error {
	// Retrieve the raw user ID string from the context, set by RefreshTokenMiddleware during authentication
	userIDRaw, ok := c.Locals("user_id").(string)
	// Check if the user ID is missing or not a string, indicating the user isn’t authenticated
	if !ok || userIDRaw == "" {
		// Log a warning to track unauthorized access attempts without a valid user ID
		logger.Log.Warn("Update attempted without user ID in context")
		// Return a 401 Unauthorized response to inform the client authentication is required
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Parse the raw user ID string into a UUID for consistent and safe handling
	userID, err := uuid.Parse(userIDRaw)
	// Check if parsing the user ID into a UUID failed due to invalid format
	if err != nil {
		// Log an error with details to debug invalid user ID formats
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userIDRaw,
		}).Error("Invalid user ID format in update")
		// Return a 400 Bad Request response to indicate the user ID is malformed
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	// Generate a unique Redis key for rate limiting updates specific to this user and section
	rateKey := "update_rate:" + userID.String() + ":" + section
	// Define the maximum number of updates allowed per minute to prevent abuse
	const maxUpdates = 5
	// Attempt to retrieve the current count of update attempts from Redis for this key
	updatesCount, err := uc.Client.Get(c.Context(), rateKey).Int()
	// Check if the rate limit key doesn’t exist in Redis (first update attempt)
	if err == redis.Nil {
		// Set the update count to 0 since this is the first attempt in the time window
		updatesCount = 0
		// Check if Redis returned an unexpected error (not just a missing key)
	} else if err != nil {
		// Log a warning to track Redis issues, but proceed without rate limiting as it’s non-critical
		logger.Log.WithFields(logrus.Fields{
			"error":   err,
			"userID":  userID,
			"section": section,
		}).Warn("Failed to check update rate limit in Redis")
	}
	// Check if the number of updates exceeds the allowed maximum
	if updatesCount >= maxUpdates {
		// Log a warning to monitor users hitting the rate limit
		logger.Log.WithFields(logrus.Fields{
			"userID":  userID,
			"section": section,
		}).Warn("Update rate limit exceeded")
		// Return a 429 Too Many Requests response to throttle excessive updates
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  "Too many update attempts, please try again later",
			"status": fiber.StatusTooManyRequests,
		})
	}
	// Increment the update counter in Redis to track this attempt
	uc.Client.Incr(c.Context(), rateKey)
	// Set a 1-minute TTL on the rate limit key to reset after the time window
	uc.Client.Expire(c.Context(), rateKey, 1*time.Minute)

	// Validate the update data using the provided section-specific validation function
	validationErrors := validateFunc(updateData)
	// Check if validation returned any errors (non-nil *ErrorResponse indicates failure)
	if validationErrors != nil {
		// Log an error with details about validation failure for debugging
		logger.Log.WithFields(logrus.Fields{
			"error":   validationErrors,
			"userID":  userID,
			"section": section,
		}).Error("Update validation failed")
		// Return a 422 Unprocessable Entity response with the validation errors
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  validationErrors,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// Initialize an empty map to store the database updates
	updates := make(map[string]interface{})
	// Call the section-specific updateFields function to populate the updates map and track changed fields
	updatedFields := updateFields(nil, updateData, updates)
	// Check if there are no fields to update (all incoming fields were nil or unchanged)
	if len(updates) == 0 {
		// Fetch the current user from Redis or DB to return in the response
		user, err := uc.FetchUser(c, userID)
		// Check if fetching the user failed (e.g., not found or DB error)
		if err != nil {
			// Return the error response from fetchUser (e.g., 404 or 500)
			return err
		}
		// Log an info message to note no updates were applied
		logger.Log.WithFields(logrus.Fields{
			"userID":  userID,
			"section": section,
		}).Info("No fields provided for update")
		// Declare a variable to hold notification preferences for the "notifications" section
		var prefs *models.NotificationPrefrences
		// Check if the section is "notifications" to fetch preferences
		if section == "notifications" {
			// Fetch notification preferences for the no-op case in the notifications section
			prefs, err = uc.userSystem.NotificationPreBy("user_id = ?", userID)
			// Check if fetching preferences failed
			if err != nil && err != gorm.ErrRecordNotFound {
				// Return a 500 Internal Server Error if fetching preferences fails unexpectedly
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":  "Failed to fetch notification preferences",
					"status": fiber.StatusInternalServerError,
				})
			}
		}
		// Return a 200 OK response with the current user data, indicating no changes were made
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "No changes provided for update",
			"status":  fiber.StatusOK,
			"section": section,
			"user":    uc.BuildProfileResponse(user, section, prefs),
		})
	}

	// Declare variables to hold updated user or notification preferences
	var updatedUser *models.User
	var updatedPrefs *models.NotificationPrefrences
	// Check if the section is "notifications" to update the separate notification preferences table
	if section == "notifications" {
		// Update the notification preferences table and get the updated preferences back
		updatedPrefs, err = uc.userSystem.NotificationPreBy("user_id = ?", userID)
		// Check if fetching the current preferences failed
		if err != nil {
			// Log an error if fetching preferences fails
			logger.Log.WithFields(logrus.Fields{
				"error":   err,
				"userID":  userID,
				"section": section,
			}).Error("Failed to fetch notification preferences")
			// Return a 404 Not Found if preferences don’t exist
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error":  "Notification preferences not found",
					"status": fiber.StatusNotFound,
				})
			}
			// Return a 500 Internal Server Error for other failures
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to fetch notification preferences",
				"status": fiber.StatusInternalServerError,
			})
		}
		// Update the preferences with provided data
		_, err = uc.userSystem.UpdateNotificationPref("user_id = ?", userID, updates)
		// Check if updating the preferences failed
		if err != nil {
			// Log an error with details about the failure
			logger.Log.WithFields(logrus.Fields{
				"error":   err,
				"userID":  userID,
				"section": section,
			}).Error("Failed to update notification preferences in database")
			// Return a 500 Internal Server Error for update failure
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to update " + section,
				"status": fiber.StatusInternalServerError,
			})
		}
		// Fetch the updated preferences post-update
		updatedPrefs, err = uc.userSystem.NotificationPreBy("user_id = ?", userID)
		// Check if fetching updated preferences failed
		if err != nil {
			// Log an error if fetching updated preferences fails
			logger.Log.WithFields(logrus.Fields{
				"error":   err,
				"userID":  userID,
				"section": section,
			}).Error("Failed to fetch updated notification preferences")
			// Return a 500 Internal Server Error for fetch failure
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to retrieve updated " + section,
				"status": fiber.StatusInternalServerError,
			})
		}
		// Fetch the user for other sections’ consistency
		updatedUser, err = uc.userSystem.UserBy("id = ?", userID)
		// Check if fetching the user failed
		if err != nil {
			// Log an error with details about the failure
			logger.Log.WithFields(logrus.Fields{
				"error":   err,
				"userID":  userID,
				"section": section,
			}).Error("Failed to fetch user after updating notification preferences")
			// Return a 500 Internal Server Error for fetch failure
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to retrieve user after update",
				"status": fiber.StatusInternalServerError,
			})
		}
	} else {
		// Update the user in the database for other sections and get the updated user back
		updatedUser, err = uc.userSystem.UpdateUser("id = ?", userID, updates)
		// Check if the update operation failed
		if err != nil {
			// Log an error with details about the failure for debugging and monitoring
			logger.Log.WithFields(logrus.Fields{
				"error":   err,
				"userID":  userID,
				"section": section,
			}).Error("Failed to update user in database")
			// Check if the error indicates the user wasn’t found
			if err.Error() == "user not found" {
				// Return a 404 Not Found response if the user doesn’t exist
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error":  "User not found",
					"status": fiber.StatusNotFound,
				})
			}
			// Return a 500 Internal Server Error for other database failures
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to update " + section,
				"status": fiber.StatusInternalServerError,
			})
		}
	}

	// Generate a Redis key for caching the updated user profile
	userKey := "user:id:" + userID.String()
	// Marshal the updated user struct into JSON for caching in Redis
	userJSON, err := json.Marshal(updatedUser)
	// Check if marshaling the user data to JSON failed
	if err != nil {
		// Log a warning if marshaling fails, but proceed as it’s non-critical (DB is source of truth)
		logger.Log.WithFields(logrus.Fields{
			"error":   err,
			"userID":  userID,
			"section": section,
		}).Warn("Failed to marshal updated user for Redis caching")
	} else {
		// Cache the updated user profile in Redis with a 30-minute TTL for quick future access
		if err := uc.Client.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
			// Log a warning if caching fails, but proceed as it’s non-critical
			logger.Log.WithFields(logrus.Fields{
				"error":   err,
				"userID":  userID,
				"section": section,
			}).Warn("Failed to update user cache in Redis")
		}
	}

	// Log a success message with the user ID and the fields that were updated
	logger.Log.WithFields(logrus.Fields{
		"userID":         userID,
		"section":        section,
		"updated_fields": updatedFields,
	}).Info("User section updated successfully")
	// Return a 200 OK response with the updated section data, changed fields, and a success message
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":        "Section updated successfully",
		"status":         fiber.StatusOK,
		"section":        section,
		"updated_fields": updatedFields,
		"user":           uc.BuildProfileResponse(updatedUser, section, updatedPrefs),
	})
}

// fetchUser retrieves the user from Redis or DB
func (uc *UserController) FetchUser(c *fiber.Ctx, userID uuid.UUID) (*models.User, error) {
	// Generate a Redis key for the cached user profile based on the user ID
	userKey := "user:id:" + userID.String()
	// Declare a variable to hold the user struct
	var user *models.User
	// Attempt to fetch the user data from Redis using the generated key
	cachedUser, err := uc.Client.Get(c.Context(), userKey).Result()
	// Check if the user data was successfully retrieved from Redis
	if err == nil {
		// Allocate a new User struct to unmarshal the cached data into
		user = &models.User{}
		// Unmarshal the JSON data from Redis into the user struct
		if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
			// Log a warning if unmarshaling fails, indicating corrupted cache data
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Warn("Failed to unmarshal cached user from Redis")
			// Set user to nil to trigger a database fallback
			user = nil
		}
	}
	// Check if the user wasn’t found in Redis (cache miss) or unmarshaling failed
	if err == redis.Nil || user == nil {
		// Fetch the user from the database using the user ID
		user, err = uc.userSystem.UserBy("id = ?", userID)
		// Check if fetching the user from the database failed
		if err != nil {
			// Log an error with details about the database failure
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Error("Database error while fetching user")
			// Check if the error is due to the user not being found in the database
			if err == gorm.ErrRecordNotFound {
				// Return a 404 Not Found response with an appropriate error message
				return nil, c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error":  "User not found",
					"status": fiber.StatusNotFound,
				})
			}
			// Return a 500 Internal Server Error for other database issues
			return nil, c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to fetch user",
				"status": fiber.StatusInternalServerError,
			})
		}
		// Marshal the fetched user data into JSON for caching
		userJSON, err := json.Marshal(user)
		// Check if marshaling the user data failed
		if err == nil {
			// Cache the user data in Redis with a 30-minute TTL for future quick access
			uc.Client.Set(c.Context(), userKey, userJSON, 30*time.Minute)
		}
	}
	// Return the fetched user and nil error if successful
	return user, nil
}

// buildProfileResponse constructs the user profile response map based on section
func (uc *UserController) BuildProfileResponse(user *models.User, section string, prefs *models.NotificationPrefrences) fiber.Map {
	// Use a switch statement to determine which section-specific fields to return
	switch section {
	// Handle the "customization" section, returning only customization-related fields
	case "customization":
		// Return a map with customization fields from the user struct
		return fiber.Map{
			"theme_preference": user.ThemePreference,
			"base_font":        user.BaseFont,
			"site_navbar":      user.SiteNavbar,
			"content_editor":   user.ContentEditor,
			"content_mode":     user.ContentMode,
		}
	// Handle the "notifications" section, returning notification preference fields
	case "notifications":
		// Check if updated preferences were provided (from UpdateNotificationPref)
		if prefs == nil {
			// Return an empty map if preferences are missing (e.g., during no-op or error)
			return fiber.Map{}
		}
		// Return a map with notification preference fields from the updated preferences
		return fiber.Map{
			"email_on_likes":     prefs.EmailOnLikes,
			"email_on_comments":  prefs.EmailOnComments,
			"email_on_mentions":  prefs.EmailOnMentions,
			"email_on_followers": prefs.EmailOnFollower,
			"email_on_badge":     prefs.EmailOnBadge,
			"email_on_unread":    prefs.EmailOnUnread,
			"email_on_new_posts": prefs.EmailOnNewPosts,
		}
	// Handle the "account" section, returning minimal data (password not exposed)
	case "account":
		// Return an empty map for account updates, avoiding password exposure
		return fiber.Map{}
	// Default case returns the full profile response for unrecognized sections
	default:
		// Return a comprehensive map with all user profile fields except sensitive relations
		return fiber.Map{
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
	}
}

// UpdateUserCustomization updates the user’s customization settings
func (uc *UserController) UpdateUserCustomization(c *fiber.Ctx) error {
	// Define a struct to hold customization update data with validation rules
	type UpdateData struct {
		ThemePreference *string `json:"theme_preference" validate:"omitempty,oneof=Light Dark"`
		BaseFont        *string `json:"base_font" validate:"omitempty,oneof=sans-serif sans jetbrainsmono hind-siliguri comic-sans"`
		SiteNavbar      *string `json:"site_navbar" validate:"omitempty,oneof=fixed static"`
		ContentEditor   *string `json:"content_editor" validate:"omitempty,oneof=rich basic"`
		ContentMode     *int    `json:"content_mode" validate:"omitempty,oneof=1 2 3 4 5"`
	}
	// Allocate a new UpdateData struct to parse the request body into
	updateData := new(UpdateData)
	// Parse the request body strictly into the UpdateData struct, ensuring correct format
	if err := utils.StrictBodyParser(c, updateData); err != nil {
		// Return a 400 Bad Request response if parsing fails due to invalid JSON or structure
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}
	// Call the helper function to process the update for the "customization" section
	return uc.UpdateUserProfileHelper(c, "customization",
		// Pass the update data to the helper
		updateData,
		// Define a validation function using the utils validator, returning *ErrorResponse
		func(data interface{}) *utils.ErrorResponse {
			// Use the validator to check the update data against defined rules
			return utils.NewValidator().Validate(data)
		},
		// Define a function to prepare updates and track changed fields
		func(user *models.User, data interface{}, updates map[string]interface{}) []string {
			// Cast the generic data to the UpdateData type for this section
			ud := data.(*UpdateData)
			// Initialize a slice to track fields that will be updated
			var updatedFields []string
			// Check if ThemePreference is provided and add it to updates
			if ud.ThemePreference != nil {
				updates["theme_preference"] = *ud.ThemePreference
				updatedFields = append(updatedFields, "theme_preference")
			}
			// Check if BaseFont is provided and add it to updates
			if ud.BaseFont != nil {
				updates["base_font"] = *ud.BaseFont
				updatedFields = append(updatedFields, "base_font")
			}
			// Check if SiteNavbar is provided and add it to updates
			if ud.SiteNavbar != nil {
				updates["site_navbar"] = *ud.SiteNavbar
				updatedFields = append(updatedFields, "site_navbar")
			}
			// Check if ContentEditor is provided and add it to updates
			if ud.ContentEditor != nil {
				updates["content_editor"] = *ud.ContentEditor
				updatedFields = append(updatedFields, "content_editor")
			}
			// Check if ContentMode is provided and add it to updates
			if ud.ContentMode != nil {
				updates["content_mode"] = *ud.ContentMode
				updatedFields = append(updatedFields, "content_mode")
			}
			// Return the list of fields that were updated
			return updatedFields
		})
}

// UpdateUserNotificationsPref updates the user’s notification preferences
func (uc *UserController) UpdateUserNotificationsPref(c *fiber.Ctx) error {
	// Define a struct to hold notification preferences update data with optional boolean fields
	type UpdateData struct {
		EmailOnLikes    *bool `json:"email_on_likes" validate:"omitempty"`
		EmailOnComments *bool `json:"email_on_comments" validate:"omitempty"`
		EmailOnMentions *bool `json:"email_on_mentions" validate:"omitempty"`
		EmailOnFollower *bool `json:"email_on_followers" validate:"omitempty"`
		EmailOnBadge    *bool `json:"email_on_badge" validate:"omitempty"`
		EmailOnUnread   *bool `json:"email_on_unread" validate:"omitempty"`
		EmailOnNewPosts *bool `json:"email_on_new_posts" validate:"omitempty"`
	}
	// Allocate a new UpdateData struct to parse the request body into
	updateData := new(UpdateData)
	// Parse the request body strictly into the UpdateData struct, ensuring correct format
	if err := utils.StrictBodyParser(c, updateData); err != nil {
		// Return a 400 Bad Request response if parsing fails due to invalid JSON or structure
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}
	// Call the helper function to process the update for the "notifications" section
	return uc.UpdateUserProfileHelper(c, "notifications",
		// Pass the update data to the helper
		updateData,
		// Define a validation function using the utils validator, returning *ErrorResponse
		func(data interface{}) *utils.ErrorResponse {
			// Use the validator to check the update data against defined rules (all fields optional)
			return utils.NewValidator().Validate(data)
		},
		// Define a function to prepare updates and track changed fields
		func(user *models.User, data interface{}, updates map[string]interface{}) []string {
			// Cast the generic data to the UpdateData type for this section
			ud := data.(*UpdateData)
			// Initialize a slice to track fields that will be updated
			var updatedFields []string
			// Check if EmailOnLikes is provided and add it to updates
			if ud.EmailOnLikes != nil {
				updates["email_on_likes"] = *ud.EmailOnLikes
				updatedFields = append(updatedFields, "email_on_likes")
			}
			// Check if EmailOnComments is provided and add it to updates
			if ud.EmailOnComments != nil {
				updates["email_on_comments"] = *ud.EmailOnComments
				updatedFields = append(updatedFields, "email_on_comments")
			}
			// Check if EmailOnMentions is provided and add it to updates
			if ud.EmailOnMentions != nil {
				updates["email_on_mentions"] = *ud.EmailOnMentions
				updatedFields = append(updatedFields, "email_on_mentions")
			}
			// Check if EmailOnFollower is provided and add it to updates
			if ud.EmailOnFollower != nil {
				updates["email_on_followers"] = *ud.EmailOnFollower
				updatedFields = append(updatedFields, "email_on_followers")
			}
			// Check if EmailOnBadge is provided and add it to updates
			if ud.EmailOnBadge != nil {
				updates["email_on_badge"] = *ud.EmailOnBadge
				updatedFields = append(updatedFields, "email_on_badge")
			}
			// Check if EmailOnUnread is provided and add it to updates
			if ud.EmailOnUnread != nil {
				updates["email_on_unread"] = *ud.EmailOnUnread
				updatedFields = append(updatedFields, "email_on_unread")
			}
			// Check if EmailOnNewPosts is provided and add it to updates
			if ud.EmailOnNewPosts != nil {
				updates["email_on_new_posts"] = *ud.EmailOnNewPosts
				updatedFields = append(updatedFields, "email_on_new_posts")
			}
			// Return the list of fields that were updated
			return updatedFields
		})
}

// UpdateUserAccount updates the user’s account settings (password)
func (uc *UserController) UpdateUserAccount(c *fiber.Ctx) error {
	// Define a struct to hold account update data with password-specific validation rules
	type UpdateData struct {
		CurrentPassword string `json:"current_password" validate:"required,min=6"`
		Password        string `json:"password" validate:"required,min=6,eqfield=ConfirmPassword"`
		ConfirmPassword string `json:"confirm_password" validate:"required,min=6"`
	}
	// Allocate a new UpdateData struct to parse the request body into
	updateData := new(UpdateData)
	// Parse the request body strictly into the UpdateData struct, ensuring correct format
	if err := utils.StrictBodyParser(c, updateData); err != nil {
		// Return a 400 Bad Request response if parsing fails due to invalid JSON or structure
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}
	// Define a custom validation function including password verification, returning *ErrorResponse
	validateAccount := func(data interface{}) *utils.ErrorResponse {
		// Cast the generic data to the UpdateData type for this section
		ud := data.(*UpdateData)
		// Validate the update data against basic rules (e.g., min length, password match)
		if err := utils.NewValidator().Validate(ud); err != nil {
			// Return the validation errors as *ErrorResponse
			return err
		}
		// Fetch the current user from the database to verify the current password
		user, err := uc.userSystem.UserBy("id = ?", uuid.MustParse(c.Locals("user_id").(string)))
		// Check if fetching the user failed
		if err != nil {
			// Return a custom *ErrorResponse indicating the user wasn’t found
			return &utils.ErrorResponse{Errors: []utils.Error{{Field: "user", Msg: "not found"}}}
		}
		// Compare the provided current password with the stored hashed password
		if err := utils.ComparePasswords(user.Password, ud.CurrentPassword); err != nil {
			// Return a custom *ErrorResponse if the current password doesn’t match
			return &utils.ErrorResponse{Errors: []utils.Error{{Field: "current_password", Msg: "mismatch"}}}
		}
		// Return nil if validation succeeds
		return nil
	}
	// Call the helper function to process the update for the "account" section
	return uc.UpdateUserProfileHelper(c, "account",
		// Pass the update data to the helper
		updateData,
		// Use the custom validation function with password check
		validateAccount,
		// Define a function to prepare updates and track changed fields
		func(user *models.User, data interface{}, updates map[string]interface{}) []string {
			// Cast the generic data to the UpdateData type for this section
			ud := data.(*UpdateData)
			// Initialize a slice to track fields that will be updated
			var updatedFields []string
			// Hash the new password for secure storage
			password, _ := utils.HashPassword(ud.Password)
			// Add the hashed password to the updates map
			updates["password"] = password
			// Track the password field as updated
			updatedFields = append(updatedFields, "password")
			// Return the list of fields that were updated
			return updatedFields
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
