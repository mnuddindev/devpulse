package models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"gorm.io/gorm"
)

type PostAnalytics struct {
	PostID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"post_id" validate:"required"`
	ViewsCount     int       `gorm:"default:0" json:"views_count"`
	CommentsCount  int       `gorm:"default:0" json:"comments_count"`
	ReactionsCount int       `gorm:"default:0" json:"reactions_count"`
	BookmarksCount int       `gorm:"default:0" json:"bookmarks_count"`
	ReadTime       int       `gorm:"default:1" json:"read_time" validate:"min=1"`
	SharesCount    int       `gorm:"default:0" json:"shares_count"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// CreatePostAnalytics creates a new PostAnalytics instance with the given post ID.
func CreatePostAnalytics(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, pa *PostAnalytics) error {
	if pa.PostID == uuid.Nil {
		return utils.NewError(utils.ErrBadRequest.Code, "PostAnalytics requires a valid post_id")
	}

	if pa.ViewsCount < 0 {
		pa.ViewsCount = 0
	}
	if pa.CommentsCount < 0 {
		pa.CommentsCount = 0
	}
	if pa.ReactionsCount < 0 {
		pa.ReactionsCount = 0
	}
	if pa.BookmarksCount < 0 {
		pa.BookmarksCount = 0
	}
	if pa.ReadTime < 1 {
		pa.ReadTime = 1
	}

	err := gormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingPostAnalytics PostAnalytics
		if err := tx.WithContext(ctx).Where("post_id = ?", pa.PostID).First(&existingPostAnalytics).Error; err == nil {
			return utils.NewError(utils.ErrBadRequest.Code, "Post analytics already exists for this post_id")
		} else if err != gorm.ErrRecordNotFound {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check existing analytics")
		}
		if err := tx.Create(pa).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create post analytics")
		}
		return nil
	})

	if err != nil {
		return err
	}

	paData, _ := json.Marshal(pa)
	redisClient.Set(ctx, "post_analytics:"+pa.PostID.String(), paData, 10*time.Minute)
	return nil
}

// GetPostAnalytics retrieves the PostAnalytics for a given post ID.
func GetPostAnalyticsBy(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, condition string, args []interface{}) (*PostAnalytics, error) {
	var cacheKey string
	if condition == "post_id = ?" {
		cacheKey = "post_analytics:" + args[0].(string)
	}
	if cached, err := rclient.Get(ctx, cacheKey).Result(); err == nil {
		var pa PostAnalytics
		if json.Unmarshal([]byte(cached), &pa) == nil {
			return &pa, nil
		}
	}

	var pa PostAnalytics
	query := db.WithContext(ctx).Where(condition, args...)
	if err := query.First(&pa).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Post analytics not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch post analytics")
	}

	paData, _ := json.Marshal(pa)
	rclient.Set(ctx, cacheKey, paData, 10*time.Minute)

	return &pa, nil
}
