package models

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"gorm.io/gorm"
)

type TagAnalytics struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TagID            uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_tag_analytics_tag_id" json:"tag_id" validate:"required"`
	DailyViews       int       `gorm:"default:0" json:"daily_views" validate:"min=0"`
	WeeklyViews      int       `gorm:"default:0" json:"weekly_views" validate:"min=0"`
	MonthlyViews     int       `gorm:"default:0" json:"monthly_views" validate:"min=0"`
	DailyFollowers   int       `gorm:"default:0" json:"daily_followers" validate:"min=0"`
	WeeklyFollowers  int       `gorm:"default:0" json:"weekly_followers" validate:"min=0"`
	MonthlyFollowers int       `gorm:"default:0" json:"monthly_followers" validate:"min=0"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Tag Tag `gorm:"foreignKey:TagID" json:"tag" validate:"-"`
}

type TagAnalyticsOption func(*TagAnalytics)

// CreateTagAnalytics creates a new TagAnalytics instance with the given options.
func CreateTagAnalytics(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, ta *TagAnalytics) error {
	if ta.TagID == uuid.Nil {
		return errors.New("TagAnalytics requires a valid tag_id")
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing TagAnalytics
		if err := tx.Where("tag_id = ?", ta.TagID).First(&existing).Error; err == nil {
			return utils.NewError(utils.ErrBadRequest.Code, "Tag analytics already exists for this tag_id")
		} else if err != gorm.ErrRecordNotFound {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check existing analytics")
		}

		if err := tx.Create(ta).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create tag analytics")
		}
		return nil
	})
	if err != nil {
		return err
	}

	taData, _ := json.Marshal(ta)
	rclient.Set(ctx, "tag_analytics:"+ta.TagID.String(), taData, 1*time.Hour)

	return nil
}

// GetTagAnalytics retrieves the TagAnalytics for a tag.
func GetTagAnalytics(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID uuid.UUID) (*TagAnalytics, error) {
	cacheKey := "tag_analytics:" + tagID.String()
	if cached, err := rclient.Get(ctx, cacheKey).Result(); err == nil {
		var ta TagAnalytics
		if json.Unmarshal([]byte(cached), &ta) == nil {
			return &ta, nil
		}
	}

	var ta TagAnalytics
	if err := db.WithContext(ctx).Where("tag_id = ?", tagID).First(&ta).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Tag analytics not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch tag analytics")
	}

	taData, _ := json.Marshal(ta)
	rclient.Set(ctx, cacheKey, taData, 1*time.Hour)

	return &ta, nil
}
