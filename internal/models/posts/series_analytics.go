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

// GetSeriesAnalytics retrieves analytics for a series
func GetSeriesAnalytics(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, seriesID uuid.UUID) (*SeriesAnalytics, error) {
	cacheKey := "series:analytics:" + seriesID.String()
	if cached, err := rclient.Get(ctx, cacheKey).Result(); err == nil {
		var analytics SeriesAnalytics
		if json.Unmarshal([]byte(cached), &analytics) == nil {
			return &analytics, nil
		}
	}

	var analytics SeriesAnalytics
	if err := db.WithContext(ctx).Where("series_id = ?", seriesID).First(&analytics).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Series analytics not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch series analytics")
	}

	data, _ := json.Marshal(analytics)
	rclient.Set(ctx, cacheKey, data, 1*time.Hour)

	return &analytics, nil
}

// UpdateSeriesAnalytics modifies analytics using options
func UpdateSeriesAnalytics(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, seriesID uuid.UUID, options ...SeriesAnalyticsOption) (*SeriesAnalytics, error) {
	tx := db.WithContext(ctx).Begin()
	analytics, err := GetSeriesAnalytics(ctx, rclient, db, seriesID)
	if err != nil {
		return nil, err
	}

	for _, opt := range options {
		opt(analytics)
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(analytics).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update series analytics")
		}

		return nil
	})
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()

	data, _ := json.Marshal(analytics)
	rclient.Set(ctx, "series:analytics:"+seriesID.String(), data, 1*time.Hour)

	return analytics, nil
}

// DeleteSeriesAnalytics removes analytics for a series
func DeleteSeriesAnalytics(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, seriesID uuid.UUID) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("series_id = ?", seriesID).Delete(&SeriesAnalytics{})
		if result.Error != nil {
			return utils.WrapError(result.Error, utils.ErrInternalServerError.Code, "Failed to delete series analytics")
		}
		return nil
	})
	if err != nil {
		return err
	}

	rclient.Del(ctx, "series:analytics:"+seriesID.String())

	return nil
}

// SyncSeriesAnalytics aggregates metrics from series posts
func SyncSeriesAnalytics(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, seriesID uuid.UUID, postsDelta, viewsDelta int) error {
	tx := db.WithContext(ctx).Begin()
	analytics, err := GetSeriesAnalytics(ctx, rclient, db, seriesID)
	if err != nil {
		if err.Error() == utils.NewError(utils.ErrNotFound.Code, "Series analytics not found").Error() {
			return nil
		}
		return err
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		newViews := analytics.TotalViews + viewsDelta
		if newViews < 0 {
			newViews = 0
		}
		analytics.TotalViews = newViews

		if postsDelta != 0 {
			var postCount int64
			if err := tx.Model(&SeriesPost{}).Where("series_id = ?", seriesID).Count(&postCount).Error; err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to count series posts")
			}
			newCount := postCount + int64(postsDelta)
			if newCount < 0 {
				newCount = 0
			}

			type PostAnalytics struct {
				TotalReactions  int
				AverageReadTime float64
				CompletionRate  float64
			}
			var result PostAnalytics
			err := tx.Table("post_analytics").
				Joins("JOIN series_posts ON series_posts.post_id = post_analytics.post_id").
				Where("series_posts.series_id = ?", seriesID).
				Select("COALESCE(SUM(post_analytics.total_reactions), 0) as total_reactions",
					"COALESCE(AVG(post_analytics.average_read_time), 0.0) as average_read_time",
					"COALESCE(AVG(post_analytics.completion_rate), 0.0) as completion_rate").
				Scan(&result).Error
			if err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to aggregate post analytics")
			}

			analytics.TotalReactions = result.TotalReactions
			analytics.AverageReadTime = result.AverageReadTime
			analytics.CompletionRate = result.CompletionRate

			if newCount == 0 {
				analytics.TotalReactions = 0
				analytics.AverageReadTime = 0.0
				analytics.CompletionRate = 0.0
			}
		}

		if err := tx.Save(analytics).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to sync series analytics")
		}

		return nil
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	data, _ := json.Marshal(analytics)
	rclient.Set(ctx, "series:analytics:"+seriesID.String(), data, 1*time.Hour)

	return nil
}
