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

// SeriesAnalyticsOption configures a SeriesAnalytics instance
type SeriesAnalyticsOption func(*SeriesAnalytics)

// CreateSeriesAnalytics initializes analytics for a series
func CreateSeriesAnalytics(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, analytics *SeriesAnalytics) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, err := GetSeries(ctx, rclient, tx, "id = ?", []interface{}{analytics.SeriesID})
		if err != nil {
			return err
		}

		var existing SeriesAnalytics
		if err := tx.Where("series_id = ?", analytics.SeriesID).First(&existing).Error; err == nil {
			return utils.NewError(utils.ErrBadRequest.Code, "Series analytics already exists")
		} else if err != gorm.ErrRecordNotFound {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check series analytics")
		}

		if err := tx.Create(analytics).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create series analytics")
		}

		return nil
	})
	if err != nil {
		return err
	}

	data, _ := json.Marshal(analytics)
	rclient.Set(ctx, "series:analytics:"+analytics.SeriesID.String(), data, 1*time.Hour)

	return nil
}
