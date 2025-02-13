package controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/sirupsen/logrus"
)

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
	user, err := uc.userSystem.UserBy("id = $1", userId)
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
		"role":                      user.Role,
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
	if err := c.BodyParser(&updateData); err != nil {
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
	if updateData.ThemePreference != nil {
		updates["theme_preference"] = updateData.ThemePreference
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
		"role":                      uu.Role,
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
		"theme_preference":          uu.ThemePreference,
		"created_at":                uu.CreatedAt,
		"updated_at":                uu.UpdatedAt,
	}

	// Return updated user profile response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user": profileResponse,
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
	if err := c.BodyParser(&updateData); err != nil {
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

func (uc *UserController) UpdateUserNotifications(c *fiber.Ctx) error {
	return nil
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
