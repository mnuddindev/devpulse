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
	updateData := new(models.UpdateUser)
	if err := c.BodyParser(&updateData); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userid": userid,
		}).Error("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// validate updateData
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

	if err := uc.userSystem.UpdateUser("id = ?", user.ID, updates); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"model": "usermodel",
		}).Error("Update failed")
	}

	uu, err := uc.userSystem.UserBy("id = ?", userid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile",
		})
	}

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

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user": profileResponse,
	})
}
