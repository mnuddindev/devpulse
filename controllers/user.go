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
	var updateData struct {
		FirstName          *string   `json:"first_name" validate:"omitempty,min=3"`
		LastName           *string   `json:"last_name" validate:"omitempty,min=3"`
		Email              *string   `json:"email" validate:"omitempty,email"`
		Username           *string   `json:"username" validate:"omitempty,min=3"`
		AvatarUrl          string    `json:"avatar_url" validate:"omitempty,url"`
		Website            *string   `json:"website" validate:"omitempty,url,max=100"`
		Location           *string   `json:"location" validate:"omitempty,max=100"`
		Bio                *string   `json:"bio" validate:"omitempty,max=200"`
		CurrentlyLearning  *string   `json:"currently_learning" validate:"omitempty,max=200"`
		AvailableFor       *string   `json:"available_for" validate:"omitempty,max=200"`
		CurrentlyHackingOn *string   `json:"currently_hacking_on" validate:"omitempty,max=200"`
		Pronouns           *string   `json:"pronouns" validate:"omitempty,max=100"`
		JobTitle           *string   `json:"job_title" validate:"omitempty,max=100"`
		Education          *string   `json:"education" validate:"omitempty,max=100"`
		BrandColor         *string   `json:"brand_color" validate:"omitempty,max=100"`
		GithubUrl          *string   `json:"github_url" validate:"omitempty,url"`
		Skills             *[]string `json:"skills"`
		Interests          *[]string `json:"interests"`
		Badges             *[]string `json:"badges"`
		Roles              *[]string `json:"roles"`
	}
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
	var user models.User
	if err := 
	return nil
}
