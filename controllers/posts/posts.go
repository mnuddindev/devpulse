package postscontroller

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	postservices "github.com/mnuddindev/devpulse/pkg/services/posts"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/sirupsen/logrus"
)

// CreatePostRequest defines the structure for post creation request
type CreatePostRequest struct {
	Title            string     `json:"title" validate:"required,min=10,max=200"`
	Content          string     `json:"content" validate:"required,min=100"`
	Excerpt          string     `json:"excerpt" validate:"max=300"`
	Slug             string     `json:"slug" validate:"max=220"`
	FeaturedImageUrl string     `json:"featured_image_url" validate:"omitempty,url,max=500"`
	Status           string     `json:"status" validate:"required,oneof=draft published unpublished public private"`
	ContentFormat    string     `json:"content_format" validate:"oneof=markdown html"`
	CanonicalURL     string     `json:"canonical_url" validate:"omitempty,url,max=500"`
	Tags             []string   `json:"tags" validate:"max=4,dive"`
	CoAuthorIDs      []string   `json:"co_authors" validate:"max=3,dive,uuid"`
	MetaTitle        string     `json:"meta_title" validate:"max=200"`
	MetaDescription  string     `json:"meta_description" validate:"max=300"`
	SEOKeywords      string     `json:"seo_keywords" validate:"max=255"`
	Published        bool       `json:"published"`
	PublishedAt      *time.Time `json:"published_at"`
	AuthorID         uuid.UUID  `json:"author_id"`
}

type PostController struct {
	postSystem *postservices.PostSystem
}

func NewPostController(postSystem *postservices.PostSystem) *PostController {
	return &PostController{
		postSystem: postSystem,
	}
}

func (pc *PostController) BeforeCreate(post *CreatePostRequest, userid uuid.UUID) (*CreatePostRequest, error) {
	if post.AuthorID == uuid.Nil {
		post.AuthorID = userid
	}
	if post.AuthorID != userid {
		return nil, errors.New("author id does not match user id")
	}
	if post.Slug == "" {
		slug := slug.Make(post.Title)
		if err := pc.postSystem.CheckSlugAvailable(slug); err != nil {
			logger.Log.WithError(err).Warn("Slug conflict")
		}
		post.Slug = slug
	}
	if post.MetaTitle == "" {
		post.MetaTitle = post.Title
	}
	if post.MetaDescription == "" {
		post.MetaDescription = post.Excerpt
	}
	if post.CanonicalURL == "" {
		post.CanonicalURL = "/post/" + post.Slug
	}
	if post.SEOKeywords == "" {
		post.SEOKeywords = utils.JoinKeywords(post.SEOKeywords, utils.GenerateKeywords(post.Title, post.Content, 10))
	}
	if post.ContentFormat == "" {
		post.ContentFormat = "markdown"
	}

	return post, nil
}

// CreatePost creates a new post
func (pc *PostController) CreatePost(c *fiber.Ctx) error {
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

	// Get roles from c.Locals and check if the user have the permission to create a post
	roles, ok := c.Locals("roles").([]string)
	if !ok || len(roles) == 0 {
		logger.Log.Error("No roles found in request context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "User roles not provided",
			"status": fiber.StatusUnauthorized,
		})
	}

	post := new(CreatePostRequest)
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

	post, err := pc.BeforeCreate(post, userId)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error creating post")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Error creating post",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Authorization: Ensure authenticated user matches the author
	if post.AuthorID != userId {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":  "Forbidden: You can only create posts for yourself",
			"status": fiber.StatusForbidden,
		})
	}

	var coAuthorUUIDs []uuid.UUID
	for _, coAuthorID := range post.CoAuthorIDs {
		coAuthorUUID, err := uuid.Parse(coAuthorID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid co_author_id format",
			})
		}
		// Ensure co-author is not the same as the author
		if coAuthorUUID == userId {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Co-author cannot be the same as the primary author",
			})
		}
		coAuthorUUIDs = append(coAuthorUUIDs, coAuthorUUID)
	}

	posts := models.Posts{
		Title:            post.Title,
		Slug:             post.Slug,
		Content:          post.Content,
		Excerpt:          post.Excerpt,
		FeaturedImageUrl: post.FeaturedImageUrl,
		Status:           post.Status,
		ContentFormat:    post.ContentFormat,
		CanonicalURL:     post.CanonicalURL,
		AuthorID:         userId,
		Published:        post.Status == "published" || post.Status == "public",
		PublishedAt:      post.PublishedAt,
		MetaTitle:        post.MetaTitle,
		MetaDescription:  post.MetaDescription,
		SEOKeywords:      post.SEOKeywords,
		CoAuthors:        coAuthors,
	}

	// Handle tags
	if len(post.Tags) > 0 {
		var tags []models.Tag
		for _, tagName := range post.Tags {
			var tag models.Tag
			if err := pc.postSystem.Crud.DB.Where("name = ?", tagName).FirstOrCreate(&tag, models.Tag{Name: tagName, Slug: slug.Make(tagName)}).Error; err != nil {
				logger.Log.WithError(err).Error("Failed to process tags")
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to process tags",
				})
			}
			tags = append(tags, tag)
		}
		posts.Tags = tags
	}

	createdPost, err := pc.postSystem.CreatePost(&posts)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error creating post")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Error creating post",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Handle co-authors (fetch users and associate them)
	if len(coAuthorUUIDs) > 0 {
		var coAuthors []models.User
		err := pc.postSystem.Crud.GetByCondition(&coAuthors, "id IN ?", []interface{}{coAuthorUUIDs}, []string{}, "", 0, 0)
		if err != nil {
			return err
		}

		// Verify all requested co-authors exist
		if len(coAuthors) != len(coAuthorUUIDs) {
			return fiber.NewError(fiber.StatusBadRequest, "One or more co-author IDs do not exist")
		}

		// Associate co-authors with the post
		if err := pc.postSystem.Crud.AddManyToMany(&createdPost, "CoAuthores", coAuthors); err != nil {
			return err
		}
	}

	// Fetch the created post with co-authors for the response
	createdPost, err = pc.postSystem.GetPostBy("id = ?", createdPost.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve created post",
		})
	}

	// Prepare response
	coAuthorIDs := make([]uuid.UUID, len(createdPost.CoAuthors))
	for i, coAuthor := range createdPost.CoAuthors {
		coAuthorIDs[i] = coAuthor.ID
	}

	logger.Log.WithFields(logrus.Fields{
		"post_id": createdPost.ID,
		"user_id": createdPost.AuthorID,
		"title":   post.Title,
	}).Info("Post created successfully")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"post":   createdPost,
		"status": fiber.StatusCreated,
	})
}
