package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
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
		"id":               user.ID,
		"username":         user.Username,
		"first_name":       user.FirstName,
		"last_name":        user.LastName,
		"bio":              user.Bio,
		"avatar_url":       user.AvatarUrl,
		"job_title":        user.JobTitle,
		"employer":         user.Employer,
		"location":         user.Location,
		"github_url":       user.GithubUrl,
		"website":          user.Website,
		"skills":           user.Skills,
		"interests":        user.Interests,
		"is_active":        user.IsActive,
		"theme_preference": user.ThemePreference,
		"created_at":       user.CreatedAt,
		"updated_at":       user.UpdatedAt,
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user": profileResponse,
	})
}
