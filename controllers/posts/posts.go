package postscontroller

import (
	"errors"
	"time"
	"unicode/utf8"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	postservices "github.com/mnuddindev/devpulse/pkg/services/posts"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CreatePostRequest defines the structure for post creation request
type CreatePostRequest struct {
	AuthorID         uuid.UUID `json:"author_id"`
	Title            string    `json:"title" validate:"required,min=10,max=200"`
	Content          string    `json:"content" validate:"required,min=100"`
	Tags             []string  `json:"tags" validate:"max=4,dive"`
	Excerpt          string    `json:"excerpt" validate:"max=300"`
	Slug             string    `json:"slug" validate:"max=220"`
	FeaturedImageUrl string    `json:"featured_image_url" validate:"omitempty,url,max=500"`
	Status           string    `json:"status" validate:"required,oneof=draft published unpublished public private"`
	ContentFormat    string    `json:"content_format" validate:"oneof=markdown html"`
	CanonicalURL     string    `json:"canonical_url" validate:"omitempty,url,max=500"`

	Published   bool       `json:"published"`
	PublishedAt *time.Time `json:"published_at"`
}

type PostController struct {
	DB         *gorm.DB
	Client     *redis.Client
	postSystem *postservices.PostSystem
}

func NewPostController(postSystem *postservices.PostSystem, db *gorm.DB, client *redis.Client) *PostController {
	return &PostController{
		DB:         db,
		Client:     client,
		postSystem: postSystem,
	}
}

func (pc *PostController) BeforeCreate(post *CreatePostRequest, userid uuid.UUID) (*CreatePostRequest, error) {
	if post.AuthorID != userid {
		return nil, errors.New("author id does not match user id")
	}
	if post.ContentFormat == "" {
		post.ContentFormat = "markdown"
	}
	if post.PublishedAt == nil && (post.Status == "published" || post.Status == "public") {
		now := time.Now()
		post.PublishedAt = &now
	}

	if post.CanonicalURL == "" {
		baseURL := "/post/" + post.Slug
		if err := pc.postSystem.CheckCanonicalURLAvailable(baseURL); err != nil {
			post.CanonicalURL = baseURL + "-" + uuid.New().String()[:8]
		} else {
			post.CanonicalURL = baseURL
		}
	}

	return post, nil
}

func (pc *PostController) NewPost(c *fiber.Ctx) error {
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

	// Role-based status
	rolePermissions := map[string]struct {
		Status    string
		Published bool
	}{
		"member":         {"moderation", false},
		"moderator":      {"published", true},
		"author":         {"published", true},
		"trusted_member": {"published", true},
		"admin":          {"published", true},
	}
	found := false
	for _, role := range roles {
		if perm, ok := rolePermissions[role]; ok {
			post.Status = perm.Status
			post.Published = perm.Published
			if perm.Published {
				now := time.Now()
				post.PublishedAt = &now
			}
			found = true
			break
		}
	}
	if !found {
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

	// Use enhanced GenerateUniqueSlug with custom config
	if post.Slug == "" {
		post.Slug, err = utils.GenerateUniqueSlug(pc.postSystem.Crud.DB, &models.Posts{}, "slug", post.Title, utils.SlugConfig{})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  err.Error(),
				"status": fiber.StatusInternalServerError,
			})
		}
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
	}

	meta := utils.GenerateMeta(post.Title, post.Content, "devpulse")

	posts.MetaTitle = meta.MetaTitle
	if post.Excerpt != "" && utf8.RuneCountInString(post.Excerpt) <= 160 {
		posts.MetaDescription = post.Excerpt
	} else {
		posts.MetaDescription = meta.MetaDesc
	}
	posts.SEOKeywords = utils.JoinKeywords(posts.SEOKeywords, meta.Keywords)

	// Handle tags
	if len(post.Tags) > 0 {
		var tags []models.Tag
		if err := pc.postSystem.Crud.GetByCondition(&tags, "name IN ?", []interface{}{post.Tags}, []string{}, "", 0, 0); err != nil {
			logger.Log.WithError(err).Error("Failed to fetch tags")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process tags"})
		}
		existingTags := make(map[string]models.Tag)
		for _, tag := range tags {
			existingTags[tag.Name] = tag
		}
		// Process tags: use existing or create new ones
		var finalTags []models.Tag
		for _, tagName := range post.Tags {
			if tag, exists := existingTags[tagName]; exists {
				finalTags = append(finalTags, tag)
			} else {
				newTag := models.Tag{
					Name: tagName,
					Slug: slug.Make(tagName),
				}
				if err := pc.postSystem.Crud.DB.Create(&newTag).Error; err != nil {
					logger.Log.WithError(err).Error("Failed to create tag")
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error":  "Failed to create tag",
						"status": fiber.StatusInternalServerError,
					})
				}
				finalTags = append(finalTags, newTag)
			}
		}
		posts.Tags = finalTags
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

	// Fetch the created post with co-authors for the response
	createdPost, err = pc.postSystem.GetPostBy("id = ?", createdPost.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve created post",
		})
	}

	// Prepare a slice of tag names for the response (instead of full Tag objects)
	var tagNames []string
	// Iterate over the post’s tags to extract names
	for _, tag := range createdPost.Tags {
		// Append each tag name to the slice
		tagNames = append(tagNames, tag.Name)
	}

	// Create a response map with only necessary fields
	response := fiber.Map{
		"status": fiber.StatusCreated, // HTTP status code
		"post": fiber.Map{
			"id":               createdPost.ID,              // Unique identifier of the post
			"title":            createdPost.Title,           // Post title
			"slug":             createdPost.Slug,            // SEO-friendly URL slug
			"status":           createdPost.Status,          // Current status (e.g., published, moderation)
			"published":        createdPost.Published,       // Published flag
			"published_at":     createdPost.PublishedAt,     // Timestamp of publication (nullable)
			"meta_title":       createdPost.MetaTitle,       // SEO meta title
			"meta_description": createdPost.MetaDescription, // SEO meta description
			"seo_keywords":     createdPost.SEOKeywords,     // SEO keywords
			"tags":             tagNames,                    // List of tag names
		},
	}

	if createdPost.Status == "moderation" {
		response["message"] = "Post submitted for moderation"
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}
