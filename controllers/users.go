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

func (uc *UserController) UserByID(c *fiber.Ctx) error {
	userinfo, err := uuid.Parse(c.Params("userid"))
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": "Invalid user id",
		}).Error("Invalid user id")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"status":   fiber.StatusUnprocessableEntity,
			"messagee": "Inavlid user id, failed to find user",
		})
	}

	user, err := uc.userSystem.UserBy("id = ?", userinfo)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": "User not found",
		}).Error("User not found")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":   fiber.StatusNotFound,
			"messagee": "user not found",
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

func (uc *UserController) UpdateUserByID(c *fiber.Ctx) error {
	// Get user ID from context
	userid, err := uuid.Parse(c.Params("userid"))
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": "Invalid user id",
		}).Error("Invalid user id")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"status":   fiber.StatusUnprocessableEntity,
			"messagee": "Inavlid user id, failed to find user",
		})
	}

	type UpdateUser struct {
		Username           *string    `json:"username" validate:"omitempty,min=3"`
		Email              *string    `json:"email" validate:"omitempty,email"`
		Password           *string    `json:"password,omitempty" validate:"omitempty,min=6"`
		FirstName          *string    `json:"first_name" validate:"omitempty,min=3"`
		LastName           *string    `json:"last_name" validate:"omitempty,min=3"`
		Bio                *string    `json:"bio,omitempty" validate:"omitempty,max=255"`
		AvatarUrl          *string    `json:"avatar_url,omitempty" validate:"omitempty,url"`
		JobTitle           *string    `json:"job_title,omitempty" validate:"omitempty,max=100"`
		Employer           *string    `json:"employer,omitempty" validate:"omitempty,max=100"`
		Location           *string    `json:"location,omitempty" validate:"omitempty,max=100"`
		GithubUrl          *string    `json:"github_url,omitempty" validate:"omitempty,url"`
		Website            *string    `json:"website,omitempty" validate:"omitempty,url"`
		CurrentLearning    *string    `json:"current_learning,omitempty" validate:"omitempty,max=200"`
		AvailableFor       *string    `json:"available_for,omitempty" validate:"omitempty,max=200"`
		CurrentlyHackingOn *string    `json:"currently_hacking_on,omitempty" validate:"omitempty,max=200"`
		Pronouns           *string    `json:"pronouns,omitempty" validate:"omitempty,max=100"`
		Education          *string    `json:"education,omitempty" validate:"omitempty,max=100"`
		BrandColor         *string    `json:"brand_color,omitempty" validate:"omitempty,max=7"`
		IsActive           *bool      `json:"is_active"`
		IsEmailVerified    *bool      `json:"is_email_verified"`
		ThemePreference    *string    `json:"theme_preference" validate:"omitempty,oneof=Light Dark"`
		BaseFont           *string    `json:"base_font" validate:"omitempty,oneof=sans-serif sans jetbrainsmono hind-siliguri comic-sans"`
		SiteNavbar         *string    `json:"site_navbar" validate:"omitempty,oneof=fixed static"`
		ContentEditor      *string    `json:"content_editor" validate:"omitempty,oneof=rich basic"`
		ContentMode        *int       `json:"content_mode" validate:"omitempty,oneof=1 2 3 4 5"`
		UpdatedAt          *time.Time `json:"updated_at"`

		Skills    *string         `json:"skills"`
		Interests *string         `json:"interests"`
		Badges    *[]models.Badge `json:"badges"`
		Roles     *[]models.Role  `json:"roles"`
	}

	// Parse request body into updateData struct
	updateData := new(UpdateUser)
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
		}).Error("User profile update validation failed while updating")
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
	if updateData.IsActive != nil {
		updates["is_active"] = updateData.IsActive
	}
	if updateData.IsEmailVerified != nil {
		updates["is_email_verified"] = updateData.IsEmailVerified
	}
	if updateData.ThemePreference != nil {
		updates["theme_preference"] = updateData.ThemePreference
	}
	if updateData.BaseFont != nil {
		updates["base_font"] = updateData.BaseFont
	}
	if updateData.SiteNavbar != nil {
		updates["site_navbar"] = updateData.SiteNavbar
	}
	if updateData.ContentEditor != nil {
		updates["content_editor"] = updateData.ContentEditor
	}
	if updateData.ContentMode != nil {
		updates["content_mode"] = updateData.ContentMode
	}
	if updateData.Skills != nil {
		updates["skills"] = updateData.Skills
	}
	if updateData.Interests != nil {
		updates["interests"] = updateData.Interests
	}

	// Handle badges before update
	if len(*updateData.Badges) > 0 {
		var newBadges []string
		for _, badge := range *updateData.Badges {
			newBadges = append(newBadges, badge.Name)
		}
		if err := uc.userSystem.UpdateBadge(userid, newBadges); err != nil {
			logrus.Error("Failed to update badges: ", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update badges"})
		}
	}

	if len(*updateData.Roles) > 0 {
		var newRoles []string
		for _, role := range *updateData.Roles {
			newRoles = append(newRoles, role.Name)
		}
		if err := uc.userSystem.UpdateRole(userid, newRoles); err != nil {
			logrus.Error("Failed to update roles: ", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update roles"})
		}
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
		if updateData.IsActive != nil {
			user.IsActive = *updateData.IsActive
		}
		if updateData.IsEmailVerified != nil {
			user.IsEmailVerified = *updateData.IsEmailVerified
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
		if updateData.Skills != nil {
			user.Skills = *updateData.Skills
		}
		if updateData.Interests != nil {
			user.Interests = *updateData.Interests
		}
		if updateData.Badges != nil {
			user.Badges = *updateData.Badges
		}
		if updateData.Roles != nil {
			user.Roles = *updateData.Roles
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
		"is_active":            user.IsActive,
		"is_email_verified":    user.IsEmailVerified,
		"theme_preference":     user.ThemePreference,
		"base_font":            user.BaseFont,
		"site_navbar":          user.SiteNavbar,
		"content_editor":       user.ContentEditor,
		"content_mode":         user.ContentMode,
		"skills":               user.Skills,
		"interests":            user.Interests,
		"badges":               user.Badges,
		"roles":                user.Roles,
	}

	// Return updated user profile response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user": profileResponse,
	})
}
