package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/sirupsen/logrus"
)

func (uc *UserController) UserProfile(c *fiber.Ctx) error {
	userId := c.Locals("user_id").(uuid.UUID)
	if userId == uuid.Nil {
		logger.Log.WithFields(logrus.Fields{
			"error": "User ID missing from context",
		}).Warn("Unauthorized access attempt")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	// fetch user profile
	user, err := uc.userSystem.UserBy("id = ?", userId)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Database error while fetching user profile")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Internal server error",
			"status": fiber.StatusInternalServerError,
		})
	}

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

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user": profileResponse,
	})
}

func (uc *UserController) UpdateUserProfile(c *fiber.Ctx) error {
	// fetching current user id from context
	userid := c.Locals("user_id").(uuid.UUID)

	// parse request body
	updateData := new(models.User)
	if err := c.BodyParser(&updateData); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userid": userid,
		}).Error("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// validate input
	validator := utils.NewValidator()
	if err := validator.Validate(updateData); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userid": userid,
		}).Error("User profile update validation failed while registering")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// find user
	user, err := uc.userSystem.UserBy("id = ?", userid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile",
		})
	}

	if err := uc.userSystem.UpdateUser("id = ?", user.ID, updateData); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userid": user.ID,
		}).Error("Failed to update user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile",
		})
	}

	if len(updateData.Skills) > 0 {
		uc.userSystem.UpdateUserMany("Skills", updateData.Skills)
	}

	if len(updateData.Interests) > 0 {
		uc.userSystem.UpdateUserMany("Interests", updateData.Interests)
	}
	if len(updateData.Badges) > 0 {
		uc.userSystem.UpdateUserMany("Badges", updateData.Badges)
	}
	if len(updateData.Roles) > 0 {
		uc.userSystem.UpdateUserMany("Roles", updateData.Roles)
	}
	if len(updateData.Followers) > 0 {
		uc.userSystem.UpdateUserMany("Followers", updateData.Followers)
	}
	if len(updateData.Following) > 0 {
		uc.userSystem.UpdateUserMany("Following", updateData.Following)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user": user,
	})
}
