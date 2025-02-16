package controllers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/auth"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/sirupsen/logrus"
)

// Registration handles user registration
func (uc *UserController) Registration(c *fiber.Ctx) error {
	var user models.User
	if err := StrictBodyParser(c, &user); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Invalid request payload")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": err.Error(),
			"status": fiber.StatusBadRequest,
		})
	}
	validator := utils.NewValidator()
	if err := validator.Validate(user); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  user,
		}).Error("User validation failed while registering")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}
	otp, err := utils.GenerateOTP()
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"field": "OTP Generation",
		}).Error("OTP Generation failed")
	}
	user.OTP = otp
	newUser, err := uc.userSystem.CreateUser(&user)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to register user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  err.Error(),
			"status": fiber.StatusInternalServerError,
		})
	}

	utils.SendActivationEmail(otp, newUser.Email, newUser.Username)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user": fiber.Map{
			"userid":   newUser.ID,
			"username": newUser.Username,
		},
		"message": "User registered successfully!!",
	})
}

// ActiveUser verifies user by otp
func (uc *UserController) ActiveUser(c *fiber.Ctx) error {
	// parse request body
	type Body struct {
		Otp int64 `json:"otp"`
	}
	var body Body
	if err := StrictBodyParser(c, &body); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}
	fmt.Println(body)

	// validate user id
	userID, err := uuid.Parse(c.Params("userid"))
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

	// check userID is not valid
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

	// Fetch user ID
	user, err := uc.userSystem.UserBy("id = ?", userID)
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

	// validate OTP
	if body.Otp != user.OTP {
		logger.Log.WithFields(logrus.Fields{
			"userID": userID,
		}).Error("OTP mismatch")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "OTP not matched",
			"status": fiber.StatusBadRequest,
		})
	}

	// check if user already verified
	if user.IsActive {
		logger.Log.WithFields(logrus.Fields{
			"userID": userID,
		}).Error("OTP expired or already verified")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "OTP expired or already verified",
			"status": fiber.StatusBadRequest,
		})
	}

	// Activate user if not activated
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

	// return success response
	logger.Log.WithFields(logrus.Fields{
		"userID": userID,
	}).Info("User activated successfully")
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
	if err := StrictBodyParser(c, &login); err != nil {
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

	// Generate JWT tokens
	atoken, rtoken, err := auth.GenerateJWT(*user)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to generate JWT tokens")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  "Token generation failed",
			"status": fiber.StatusUnprocessableEntity,
		})
	}
	at := fiber.Cookie{
		Name:     "access_token",
		Value:    atoken,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
	}
	rt := fiber.Cookie{
		Name:     "refresh_token",
		Value:    rtoken,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: true,
	}
	c.Cookie(&at)
	c.Cookie(&rt)
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
		"id":                        user.ID,
		"username":                  user.Username,
		"first_name":                user.FirstName,
		"last_name":                 user.LastName,
		"email":                     user.Email,
		"bio":                       user.Bio,
		"avatar_url":                user.AvatarUrl,
		"job_title":                 user.JobTitle,
		"employer":                  user.Employer,
		"location":                  user.Location,
		"github_url":                user.GithubUrl,
		"website":                   user.Website,
		"is_email_verified":         user.IsEmailVerified,
		"posts_count":               user.PostsCount,
		"comments_count":            user.CommentsCount,
		"likes_count":               user.LikesCount,
		"bookmark_count":            user.BookmarksCount,
		"last_seen":                 user.LastSeen,
		"skills":                    user.Skills,
		"interests":                 user.Interests,
		"badges":                    user.Badges,
		"roles":                     user.Roles,
		"followers":                 user.Followers,
		"following":                 user.Following,
		"notifications":             user.Notifications,
		"notifications_preferences": user.NotificationsPreferences,
		"is_active":                 user.IsActive,
		"theme_preference":          user.ThemePreference,
		"created_at":                user.CreatedAt,
		"updated_at":                user.UpdatedAt,
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
	if err := StrictBodyParser(c, &updateData); err != nil {
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

	// Update user in the database
	if err := uc.userSystem.UpdateUser("id = ?", user.ID, updates); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"model": "usermodel",
		}).Error("Update failed")
	}

	// Fetch updated user profile from the database
	uu, err := uc.userSystem.UserBy("id = ?", userid)
	if err != nil {
		// Return internal server error status
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile",
		})
	}

	// Prepare updated user profile response
	profileResponse := fiber.Map{
		"id":                        uu.ID,
		"username":                  uu.Username,
		"first_name":                uu.FirstName,
		"last_name":                 uu.LastName,
		"email":                     uu.Email,
		"bio":                       uu.Bio,
		"avatar_url":                uu.AvatarUrl,
		"job_title":                 uu.JobTitle,
		"employer":                  uu.Employer,
		"location":                  uu.Location,
		"github_url":                uu.GithubUrl,
		"website":                   uu.Website,
		"is_email_verified":         uu.IsEmailVerified,
		"posts_count":               uu.PostsCount,
		"comments_count":            uu.CommentsCount,
		"likes_count":               uu.LikesCount,
		"bookmark_count":            uu.BookmarksCount,
		"last_seen":                 uu.LastSeen,
		"skills":                    uu.Skills,
		"interests":                 uu.Interests,
		"badges":                    uu.Badges,
		"roles":                     uu.Roles,
		"followers":                 uu.Followers,
		"following":                 uu.Following,
		"notifications":             uu.Notifications,
		"notifications_preferences": uu.NotificationsPreferences,
		"is_active":                 uu.IsActive,
		"created_at":                uu.CreatedAt,
		"updated_at":                uu.UpdatedAt,
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
	if err := StrictBodyParser(c, &updateData); err != nil {
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
			"error": "Notification Preferences not found",
		}).Warn("Unauthorized access attempt")
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
	if err := StrictBodyParser(c, &updateData); err != nil {
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
	if err := StrictBodyParser(c, &updateData); err != nil {
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
