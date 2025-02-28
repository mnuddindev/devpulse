package users

import (
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

// Registration handles user registration and assigns the pre-seeded "member" role
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
		// Return a 500 Internal Server Error if OTP generation fails (optional improvement)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to generate OTP",
			"status": fiber.StatusInternalServerError,
		})
	}
	// Assign the generated OTP to the user struct
	user.OTP = otp

	// Fetch the pre-seeded "member" role from the database
	var memberRole models.Role
	err = uc.userSystem.Crud.GetByCondition(&memberRole, "name = ?", []interface{}{"member"}, []string{}, "", 0, 0)
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
		// Log an error with details if user creation fails
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Failed to register user")
		// Return a 500 Internal Server Error with the error message
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  err.Error(),
			"status": fiber.StatusInternalServerError,
		})
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

// ActiveUser verifies and activates a user by OTP, using Redis for performance
func (uc *UserController) ActiveUser(c *fiber.Ctx) error {
	// Define a struct to parse the JSON request body containing the OTP
	type Body struct {
		Otp int64 `json:"otp"`
	}
	// Declare a variable to hold the parsed request body
	var body Body
	// Parse the request body strictly into the Body struct
	if err := utils.StrictBodyParser(c, &body); err != nil {
		// Log an error with details if parsing fails
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body")
		// Return a 400 Bad Request response with an error message
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	// Parse the user ID from the URL parameter "userid"
	userID, err := uuid.Parse(c.Params("userid"))
	// Check if the user ID is invalid or empty (uuid.Nil)
	if err != nil || userID == uuid.Nil {
		// Log an error with the invalid user ID
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Invalid user ID")
		// Return a 404 Not Found response for an invalid user ID
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "User not found",
			"status": fiber.StatusNotFound,
		})
	}

	// Check if the user ID is the zero UUID (additional validation)
	if userID.String() == "00000000-0000-0000-0000-000000000000" {
		// Log an error indicating the user ID is invalid
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("User not found")
		// Return a 404 Not Found response for a zero UUID
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errors": "user not found",
			"status": fiber.StatusNotFound,
		})
	}

	// Generate a Redis key for the user’s cached data (e.g., "user:uuid")
	userKey := "user:" + userID.String()
	// Attempt to fetch the user’s OTP from Redis
	cachedOTP, err := uc.Client.Get(c.Context(), userKey+":otp").Int64()
	// Declare a variable to hold the user struct
	var user *models.User
	// Check if the OTP is cached in Redis
	if err != nil {
		// Validate the OTP from the request against the cached OTP
		if body.Otp != cachedOTP {
			// Log an error if the OTP doesn’t match
			logger.Log.WithFields(logrus.Fields{
				"userID": userID,
			}).Error("OTP mismatch")
			// Return a 400 Bad Request response for an incorrect OTP
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "OTP not matched",
				"status": fiber.StatusBadRequest,
			})
		}
		// Fetch the user from the database only if OTP matches (to check IsActive)
		user, err = uc.userSystem.UserBy("id = ?", userID)
		// Check if fetching the user failed
		if err != nil {
			// Log an error with details if the user isn’t found
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Error("Failed to fetch user by ID")
			// Return a 404 Not Found response if the user doesn’t exist
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":  "User not found",
				"status": fiber.StatusNotFound,
			})
		}
	} else if err == redis.Nil {
		// If Redis cache misses (OTP not found), fetch the user from the database
		user, err = uc.userSystem.UserBy("id = ?", userID)
		// Check if fetching the user failed
		if err != nil {
			// Log an error with details if the user isn’t found
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Error("Failed to fetch user by ID")
			// Return a 404 Not Found response if the user doesn’t exist
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":  "User not found",
				"status": fiber.StatusNotFound,
			})
		}
		// Validate the OTP from the request against the database OTP
		if body.Otp != user.OTP {
			// Log an error if the OTP doesn’t match
			logger.Log.WithFields(logrus.Fields{
				"userID": userID,
			}).Error("OTP mismatch")
			// Return a 400 Bad Request response for an incorrect OTP
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "OTP not matched",
				"status": fiber.StatusBadRequest,
			})
		}
		// Cache the OTP in Redis with a 15-minute TTL after DB fetch
		if err := uc.Client.Set(c.Context(), userKey+":otp", user.OTP, 15*time.Minute).Err(); err != nil {
			// Log a warning if caching fails (non-critical, proceed anyway)
			logger.Log.WithFields(logrus.Fields{
				"error":  err,
				"userID": userID,
			}).Warn("Failed to cache OTP in Redis")
		}
	} else {
		// Log an error if Redis access fails for another reason
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Redis error fetching OTP")
		// Return a 500 Internal Server Error for Redis issues
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Internal server error",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Check if the user is already activated
	if user.IsActive {
		// Log an error if the user is already verified
		logger.Log.WithFields(logrus.Fields{
			"userID": userID,
		}).Error("OTP expired or already verified")
		// Return a 400 Bad Request response if the user is already active
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "OTP expired or already verified",
			"status": fiber.StatusBadRequest,
		})
	}

	// Activate the user by updating the database
	if err := uc.userSystem.ActiveUser(userID); err != nil {
		// Log an error with details if activation fails
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Failed to activate user")
		// Return a 500 Internal Server Error if activation fails
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to activate account",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Invalidate the OTP cache in Redis after successful activation
	if err := uc.Client.Del(c.Context(), userKey+":otp").Err(); err != nil {
		// Log a warning if cache deletion fails (non-critical, proceed anyway)
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Warn("Failed to delete OTP from Redis cache")
	}

	// Log a success message for the activation
	logger.Log.WithFields(logrus.Fields{
		"userID": userID,
	}).Info("User activated successfully")
	// Return a 200 OK response with user details
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

// Login make sure to checks and let users to login
func (uc *UserController) Login(c *fiber.Ctx) error {
	type Login struct {
		Email    string `json:"email" validate:"required,email,min=5"`
		Password string `json:"password" validate:"required,min=6"`
	}
	// parse request body
	var login Login
	if err := utils.StrictBodyParser(c, &login); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	// validate email password
	validator := utils.NewValidator()
	if err := validator.Validate(login); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  login,
		}).Error("User validation failed while registering")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// fetch user by email
	user, err := uc.userSystem.UserBy("email = ?", login.Email)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"email": login.Email,
		}).Error("Failed to fetch user by email")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "User not found",
			"status": fiber.StatusNotFound,
		})
	}

	// compare user password
	if err := utils.ComparePasswords(user.Password, login.Password); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"email": login.Email,
		}).Error("Password mismatch")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Email or password not matched",
			"status": fiber.StatusUnauthorized,
		})
	}

	// check if user is activated
	if !user.IsActive {
		logger.Log.WithFields(logrus.Fields{
			"email": login.Email,
		}).Error("User not verified")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Verify your account first",
			"status": fiber.StatusUnauthorized,
		})
	}

	var roleIDs []uuid.UUID
	var permissions []string
	for _, role := range user.Roles {
		roleIDs = append(roleIDs, role.ID)
		for _, perm := range role.Permissions {
			permissions = append(permissions, perm.Name)
		}
	}

	// Generate JWT tokens
	atoken, rtoken, err := auth.GenerateJWT(*user, roleIDs)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to generate JWT tokens")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  "Token generation failed",
			"status": fiber.StatusUnprocessableEntity,
		})
	}
	// Set cookies explicitly
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    atoken,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Strict",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    rtoken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   false, // Set to true in production
		SameSite: "Strict",
	})
	logger.Log.WithFields(logrus.Fields{
		"email": login.Email,
	}).Info("User logged in successfully")
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
