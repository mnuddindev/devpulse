package models

import (
	"time"

	"github.com/google/uuid"
)

type SeriesAnalytics struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SeriesID        uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_series_analytics" json:"series_id" validate:"required"`
	TotalViews      int       `gorm:"default:0" json:"total_views" validate:"min=0"`
	TotalReactions  int       `gorm:"default:0" json:"total_reactions" validate:"min=0"`
	AverageReadTime float64   `gorm:"default:0.0" json:"average_read_time" validate:"min=0.0"`
	CompletionRate  float64   `gorm:"default:0.0" json:"completion_rate" validate:"min=0.0,max=1.0"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
