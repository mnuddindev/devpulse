package postscontroller

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	postservices "github.com/mnuddindev/devpulse/pkg/services/posts"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/sirupsen/logrus"
)

type PostController struct {
	postSystem *postservices.PostSystem
}

func NewPostController(postSystem *postservices.PostSystem) *PostController {
	return &PostController{
		postSystem: postSystem,
	}
}

// CreatePost creates a new post
func (pc *PostController) CreatePost(c *fiber.Ctx) error {
	post := new(models.Posts)
	if err := utils.StrictBodyParser(c, post); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Invalid request payload")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": err.Error(),
			"status": fiber.StatusBadRequest,
		})
	}

	validator := utils.NewValidator()
	if err := validator.Validate(post); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"post":  post,
		}).Error("Invalid request payload")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  err,
			"status": fiber.StatusBadRequest,
		})
	}

	// Get roles from c.Locals and check if the user have the permission to create a post
	roles, ok := c.Locals("roles").([]string)
	if !ok || len(roles) == 0 {
		logger.Log.Error("No roles found in request context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "User roles not provided",
			"status": fiber.StatusUnauthorized,
		})
	}

	if utils.Contains(roles, "member") && !utils.Contains(roles, "moderator") && !utils.Contains(roles, "author") && !utils.Contains(roles, "trusted_member") && !utils.Contains(roles, "admin") {
		// Member-only role: Post goes to moderation
		post.Status = "moderation"
		post.Published = false
	} else if utils.Contains(roles, "moderator") || utils.Contains(roles, "author") || utils.Contains(roles, "trusted_member") || utils.Contains(roles, "admin") {
		// Higher roles: Post is published directly
		post.Status = "published"
		post.Published = true
		post.PublishedAt = &time.Time{} // Set to current time; adjust if needed
		*post.PublishedAt = time.Now()
	} else {
		// No recognized role: Deny creation
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":  "Insufficient permissions to create post",
			"status": fiber.StatusForbidden,
		})
	}

	createdPost, err := pc.postSystem.CreatePost(post)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error creating post")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Error creating post",
			"status": fiber.StatusInternalServerError,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"post":   createdPost,
		"status": fiber.StatusCreated,
	})
}
