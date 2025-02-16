package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
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
