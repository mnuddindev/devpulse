package models

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"gorm.io/gorm"
)

type Tag struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name             string    `gorm:"size:30;not null;uniqueIndex:idx_tag_name" json:"name" validate:"required,min=2,max=30,alphanumunicode"`
	Slug             string    `gorm:"size:35;not null;uniqueIndex:idx_tag_slug" json:"slug" validate:"required,max=35,customSlug"`
	Description      string    `gorm:"size:200" json:"description" validate:"omitempty,max=200"`
	ShortDescription string    `gorm:"size:100" json:"short_description" validate:"omitempty,max=100"`
	IconURL          string    `gorm:"size:500" json:"icon_url" validate:"omitempty,url,max=500"`
	BackgroundURL    string    `gorm:"size:500" json:"background_url" validate:"omitempty,url,max=500"`
	TextColor        string    `gorm:"size:7" json:"text_color" validate:"omitempty,hexcolor,max=7"`
	BackgroundColor  string    `gorm:"size:7" json:"background_color" validate:"omitempty,hexcolor,max=7"`
	IsApproved       bool      `gorm:"default:false;index" json:"is_approved"`
	IsFeatured       bool      `gorm:"default:false;index" json:"is_featured"`
	IsModerated      bool      `gorm:"default:false;index" json:"is_moderated"`
	PostsCount       int       `gorm:"default:0" json:"posts_count" validate:"min=0"`
	FollowersCount   int       `gorm:"default:0" json:"followers_count" validate:"min=0"`
	Rules            string    `gorm:"size:500" json:"rules" validate:"omitempty,max=500"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Posts      []Posts       `gorm:"many2many:post_tags;" json:"posts" validate:"-"`
	Followers  []user.User   `gorm:"many2many:tag_followers;" json:"followers" validate:"-"`
	Moderators []user.User   `gorm:"many2many:tag_moderators;" json:"moderators" validate:"-"`
	Analytics  *TagAnalytics `gorm:"foreignKey:TagID" json:"analytics" validate:"-"`
}

// TagOptions defines a function type that takes a pointer to a Tag and modifies it.
type TagOption func(*Tag)

// CreateTag creates a new Tag instance with the provided options.
func CreateTag(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tag *Tag) error {
	tag.Name = strings.TrimSpace(tag.Name)
	tag.Slug = strings.ToLower(strings.TrimSpace(tag.Slug))
	if tag.Name == "" || tag.Slug == "" {
		return utils.NewError(utils.ErrBadRequest.Code, "Tag name and slug are required")
	}

	if !tag.IsApproved {
		tag.IsApproved = false
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingByName Tag
		if err := tx.Where("name = ?", tag.Name).First(&existingByName).Error; err == nil {
			return utils.NewError(utils.ErrBadRequest.Code, "Tag name already exists")
		} else if err != gorm.ErrRecordNotFound {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check tag name")
		}

		if err := tx.Create(tag).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create tag")
		}

		return nil
	})
	if err != nil {
		return err
	}

	tagData, _ := json.Marshal(tag)
	rclient.Set(ctx, "tag:"+tag.ID.String(), tagData, 24*time.Hour)
	rclient.Set(ctx, "tag:slug:"+tag.Slug, tagData, 24*time.Hour)

	return nil
}

// GetTagBy retrieves a tags
func GetTagBy(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, condition string, args []interface{}) (*Tag, error) {
	cacheKey := ""
	switch condition {
	case "id = ?":
		cacheKey = "tag:" + args[0].(uuid.UUID).String()
	case "slug = ?":
		cacheKey = "tag:slug:" + args[0].(string)
	}
	if cacheKey != "" {
		if cached, err := rclient.Get(ctx, cacheKey).Result(); err == nil {
			var tag Tag
			if json.Unmarshal([]byte(cached), &tag) == nil {
				return &tag, nil
			}
		}
	}

	var tag Tag
	if err := db.WithContext(ctx).Where(condition, args...).Preload("Moderators").Preload("Followers").Preload("Posts").Preload("Analytics").First(&tag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Tag not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch tag")
	}

	if cacheKey != "" {
		tagData, _ := json.Marshal(tag)
		rclient.Set(ctx, cacheKey, tagData, 24*time.Hour)
	}

	return &tag, nil
}

// UpdateTag updates a tag in the database and cache
func UpdateTag(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID uuid.UUID, options ...TagOption) (*Tag, error) {
	tx := db.WithContext(ctx).Begin()
	tag, err := GetTagBy(ctx, rclient, db, "id = ?", []interface{}{tagID})
	if err != nil {
		return nil, err
	}

	originalSlug := tag.Slug
	for _, opt := range options {
		opt(tag)
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		// Check name uniqueness if changed
		if tag.Name != "" && tag.Name != originalSlug {
			var existing Tag
			if err := tx.Where("name = ? AND id != ?", tag.Name, tagID).First(&existing).Error; err == nil {
				return utils.NewError(utils.ErrBadRequest.Code, "Tag name already taken")
			} else if err != gorm.ErrRecordNotFound {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check tag name")
			}
		}

		// Save tag
		if err := tx.Save(tag).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update tag")
		}

		return nil
	})
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()

	// Update caches
	tagData, _ := json.Marshal(tag)
	rclient.Set(ctx, "tag:"+tag.ID.String(), tagData, 24*time.Hour)
	rclient.Set(ctx, "tag:slug:"+tag.Slug, tagData, 24*time.Hour)
	if tag.Slug != originalSlug {
		rclient.Del(ctx, "tag:slug:"+originalSlug)
	}

	return tag, nil
}

// DeleteTag soft-deletes a tag
func DeleteTag(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID uuid.UUID) error {
	tx := db.WithContext(ctx).Begin()
	tag, err := GetTagBy(ctx, rclient, db, "id = ?", []interface{}{tagID})
	if err != nil {
		return err
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(tag).Association("Moderators").Clear(); err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to clear tag moderators")
		}

		if err := tx.Model(tag).Association("Followers").Clear(); err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to clear tag followers")
		}

		if err := tx.Delete(tag).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to delete tag")
		}

		return nil
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	rclient.Del(ctx, "tag:"+tagID.String(), "tag:slug:"+tag.Slug)

	return nil
}
