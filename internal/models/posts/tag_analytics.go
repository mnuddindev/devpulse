package models

import (
	"time"

	"github.com/google/uuid"
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
