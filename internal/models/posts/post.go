package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Posts struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Title            string     `gorm:"size:200;not null;index:idx_post_title" json:"title" validate:"required,min=10,max=200"`
	Slug             string     `gorm:"size:220;not null;uniqueIndex:idx_post_slug" json:"slug" validate:"required,max=220,customSlug"`
	Content          string     `gorm:"type:text;not null" json:"content" validate:"required,min=100"`
	Excerpt          string     `gorm:"size:300" json:"excerpt" validate:"omitempty,max=300"`
	FeaturedImageURL string     `gorm:"size:500" json:"featured_image_url" validate:"omitempty,url,max=500"`
	Published        bool       `gorm:"default:false;index" json:"published"`
	PublishedAt      *time.Time `gorm:"index:idx_post_published_at" json:"published_at" validate:"omitempty"`
	Status           string     `gorm:"type:varchar(20);default:'draft';index" json:"status" validate:"required,oneof=draft published unpublished public private"`
	PublishingStatus string     `gorm:"type:varchar(50);default:'draft'" json:"publishing_status"`
	ContentFormat    string     `gorm:"size:20;default:'markdown'" json:"content_format" validate:"oneof=markdown html"`
	CanonicalURL     string     `gorm:"size:500" json:"canonical_url" validate:"omitempty,url,max=500"`

	// SEO & Social Metadata
	MetaTitle          string `gorm:"size:200" json:"meta_title" validate:"omitempty,max=200"`
	MetaDescription    string `gorm:"size:300" json:"meta_description" validate:"omitempty,max=300"`
	SEOKeywords        string `gorm:"size:255" json:"seo_keywords" validate:"omitempty,max=255"`
	OGTitle            string `gorm:"size:200" json:"og_title" validate:"omitempty,max=200"`
	OGDescription      string `gorm:"size:300" json:"og_description" validate:"omitempty,max=300"`
	OGImageURL         string `gorm:"size:500" json:"og_image_url" validate:"omitempty,url,max=500"`
	TwitterTitle       string `gorm:"size:200" json:"twitter_title" validate:"omitempty,max=200"`
	TwitterDescription string `gorm:"size:300" json:"twitter_description" validate:"omitempty,max=300"`
	TwitterImageURL    string `gorm:"size:500" json:"twitter_image_url" validate:"omitempty,url,max=500"`

	// Collaboration & Review System
	AuthorID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_post_author" json:"author_id" validate:"required"`
	SeriesID       *uuid.UUID `gorm:"type:uuid;index:idx_post_series" json:"series_id" validate:"omitempty"`
	EditedAt       *time.Time `gorm:"index" json:"edited_at" validate:"omitempty"`
	LastEditedByID *uuid.UUID `gorm:"type:uuid" json:"last_edited_by_id" validate:"omitempty"`
	NeedsReview    bool       `gorm:"default:false;index" json:"needs_review"`
	ReviewedByID   *uuid.UUID `gorm:"type:uuid" json:"reviewed_by_id" validate:"omitempty"`
	ReviewedAt     *time.Time `gorm:"index" json:"reviewed_at" validate:"omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Author        user.User      `gorm:"foreignKey:AuthorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"author" validate:"-"`
	Series        *Series        `gorm:"foreignKey:SeriesID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"series" validate:"-"`
	Tags          []Tag          `gorm:"many2many:post_tags;" json:"tags" validate:"max=4,dive"`
	Comments      []Comment      `gorm:"foreignKey:PostID" json:"comments" validate:"-"`
	Reactions     []Reaction     `gorm:"foreignKey:ReactableID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"reactions" validate:"-"`
	Bookmarks     []Bookmark     `gorm:"foreignKey:PostID" json:"bookmarks" validate:"-"`
	Mentions      []user.User    `gorm:"many2many:post_mentions;" json:"mentions" validate:"max=5,dive"`
	CoAuthors     []user.User    `gorm:"many2many:post_co_authors;" json:"co_authors" validate:"max=3,dive"`
	PostAnalytics *PostAnalytics `gorm:"foreignKey:PostID" json:"analytics" validate:"-"`
}

// PostsOption configures a Post.
type PostsOption func(*Posts)

// CreatePost creates a new post in the database
func CreatePost(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, post *Posts, opts ...PostsOption) error {
	if post.Status == "" {
		post.Status = "draft"
	}
	if post.ContentFormat == "" {
		post.ContentFormat = "markdown"
	}
	if post.Published && post.PublishedAt == nil {
		now := time.Now()
		post.PublishedAt = &now
	}

	post.Title = strings.TrimSpace(post.Title)
	post.Slug = strings.TrimSpace(post.Slug)
	post.Content = strings.TrimSpace(post.Content)
	if post.AuthorID == uuid.Nil || post.Title == "" || post.Slug == "" || post.Content == "" {
		return utils.NewError(utils.ErrBadRequest.Code, "Required fields missing: author_id, title, slug, content")
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var author user.User
		key := "user:" + post.AuthorID.String()
		autho, err := rclient.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				author, err := user.GetUserBy(ctx, rclient, tx, "id = ?", []interface{}{post.AuthorID})
				if err != nil {
					return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch author")
				}
				authorData, _ := json.Marshal(author)
				rclient.Set(ctx, key, authorData, 10*time.Minute)
			} else {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch author from cache")
			}
		} else {
			if err := json.Unmarshal([]byte(autho), &author); err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to unmarshal author data")
			}
		}

		validStatuses := map[string]bool{"draft": true, "published": true, "unpublished": true, "public": true, "private": true}
		if !validStatuses[post.Status] {
			return utils.NewError(utils.ErrBadRequest.Code, "Invalid post status")
		}

		if author.Role.Name == "member" {
			post.PublishingStatus = "moderation"
			post.Published = false
			post.PublishedAt = nil
		} else {
			post.PublishingStatus = "published"
			post.Published = (post.Status == "published" || post.Status == "public")
			if post.Published && post.PublishedAt == nil {
				now := time.Now()
				post.PublishedAt = &now
			} else if !post.Published {
				post.PublishedAt = nil
			}
		}

		if err := tx.WithContext(ctx).Create(post).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create post")
		}

		analytics := &PostAnalytics{PostID: post.ID}
		if err := CreatePostAnalytics(ctx, rclient, tx, analytics); err != nil {
			return err
		}
		post.PostAnalytics = analytics

		for _, tag := range post.Tags {
			if err := IncrementTagCounts(ctx, rclient, tx, tag.ID, 1, 0); err != nil {
				return err
			}
			if err := SyncTagAnalytics(ctx, rclient, tx, tag.ID, 1, 0); err != nil { // Initial view on creation
				return err
			}
		}

		return user.UpdateUserStats(ctx, rclient, tx, post.AuthorID, user.WithPostsCount(1))
	})

	if err != nil {
		return err
	}

	postData, _ := json.Marshal(post)
	rclient.Set(ctx, "post:"+post.ID.String(), postData, 10*time.Minute)
	rclient.Del(ctx, "public_post:"+post.Slug)

	return nil
}

// GetPostsBy retrieves a post by condition, with optional preloading of relationships.
func GetPostsBy(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, condition string, args []interface{}, preload ...string) (*Posts, error) {
	var post Posts
	query := db.WithContext(ctx).Where(condition, args...)
	for _, rel := range preload {
		query = query.Preload(rel)
	}
	if err := query.First(&post).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Post not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch post")
	}

	return &post, nil
}

// GetPosts retrieves multiple users with pagination and optional filters.
func GetPosts(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, page, limit int, filters ...string) ([]Posts, error) {
	cacheKey := fmt.Sprintf("posts:page:%d:limit:%d", page, limit)
	if cached, err := redisClient.Get(ctx, cacheKey).Result(); err == nil {
		var posts []Posts
		if json.Unmarshal([]byte(cached), &posts) == nil {
			return posts, nil
		}
	}

	var posts []Posts
	query := gormDB.WithContext(ctx).Limit(limit).Offset((page - 1) * limit).Order("created_at desc")

	for _, filter := range filters {
		query = query.Where(filter)
	}

	if err := query.Find(&posts).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "No posts found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch posts")
	}

	return posts, nil
}

// UpdatePost updates a post in the database
func UpdatePost(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, post *Posts, opts ...PostsOption) (*Posts, error) {
	tx := db.WithContext(ctx).Begin()
	post, err := GetPostsBy(ctx, rclient, tx, "id = ?", []interface{}{post.ID}, "Mentions", "CoAuthors", "PostAnalytics")
	if err != nil {
		return nil, err
	}

	originalSlug := post.Slug
	originalTags := post.Tags
	for _, opt := range opts {
		opt(post)
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		if post.Slug != originalSlug && post.Slug != "" {
			var existing Posts
			if err := tx.Where("slug = ?", post.Slug).First(&existing).Error; err == nil {
				return utils.NewError(utils.ErrBadRequest.Code, "Slug already taken")
			} else if err != gorm.ErrRecordNotFound {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check slug")
			}
		}

		originalTagIDs := make(map[uuid.UUID]bool)
		for _, tag := range originalTags {
			originalTagIDs[tag.ID] = true
		}

		newTagIDs := make(map[uuid.UUID]bool)
		for _, tag := range post.Tags {
			newTagIDs[tag.ID] = true
		}

		for tagID := range originalTagIDs {
			if !newTagIDs[tagID] {
				if err := IncrementTagCounts(ctx, rclient, tx, tagID, -1, 0); err != nil {
					return err
				}
			}
		}

		for tagID := range newTagIDs {
			if !originalTagIDs[tagID] {
				if err := IncrementTagCounts(ctx, rclient, tx, tagID, 1, 0); err != nil {
					return err
				}
			}
		}

		if err := tx.Save(post).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update post")
		}

		return nil
	})
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()

	postData, _ := json.Marshal(post)
	rclient.Set(ctx, "post:"+post.ID.String(), postData, 10*time.Minute)
	rclient.Del(ctx, "public_post:"+originalSlug)
	if post.Slug != originalSlug {
		rclient.Del(ctx, "public_post:"+post.Slug)
	}

	return post, nil
}

// DeletePost deletes a post from the database
func DeletePost(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, postID uuid.UUID) error {
	tx := db.WithContext(ctx).Begin()
	post, err := GetPostsBy(ctx, rclient, tx, "id = ?", []interface{}{postID}, "Mentions", "CoAuthors", "PostAnalytics")
	if err != nil {
		return err
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		if err := DeletePostAnalytics(ctx, rclient, db, postID); err != nil {
			return err
		}

		if err := tx.Model(post).Association("Mentions").Clear(); err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to clear post mentions")
		}

		if err := tx.Model(post).Association("CoAuthors").Clear(); err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to clear post co-authors")
		}

		for _, tag := range post.Tags {
			if err := IncrementTagCounts(ctx, rclient, tx, tag.ID, -1, 0); err != nil {
				return err
			}
		}

		if err := tx.Delete(post).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to delete post")
		}

		return user.UpdateUserStats(ctx, rclient, tx, post.AuthorID, user.WithPostsCount(-1))
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	rclient.Del(ctx, "post:"+post.ID.String())
	rclient.Del(ctx, "public_post:"+post.Slug)

	return nil
}
