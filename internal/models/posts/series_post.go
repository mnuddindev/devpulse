package models

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"gorm.io/gorm"
)

type SeriesPost struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SeriesID uuid.UUID `gorm:"type:uuid;not null;index:idx_series_post_series" json:"series_id" validate:"required"`
	PostID   uuid.UUID `gorm:"type:uuid;not null;index:idx_series_post_post" json:"post_id" validate:"required"`
	Position int       `gorm:"not null;index:idx_series_position" json:"position" validate:"min=1"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Post   Posts  `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"post" validate:"-"`
	Series Series `gorm:"foreignKey:SeriesID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"series" validate:"-"`
}

// SeriesPostOption configures a SeriesPost instance
type SeriesPostOption func(*SeriesPost)

// CreateSeriesPost adds a post to a series
func CreateSeriesPost(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, seriesPost *SeriesPost) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		series, err := GetSeries(ctx, rclient, tx, "id = ?", []interface{}{seriesPost.SeriesID})
		if err != nil {
			return err
		}

		_, err = GetPostsBy(ctx, rclient, tx, "id = ?", []interface{}{seriesPost.PostID})
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return utils.NewError(utils.ErrNotFound.Code, "Post not found")
			}
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch post")
		}

		var existing SeriesPost
		if err := tx.Where("series_id = ? AND post_id = ?", seriesPost.SeriesID, seriesPost.PostID).First(&existing).Error; err == nil {
			return utils.NewError(utils.ErrBadRequest.Code, "Post already in series")
		} else if err != gorm.ErrRecordNotFound {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check series post")
		}

		var maxPosition int
		if err := tx.Model(&SeriesPost{}).Where("series_id = ?", seriesPost.SeriesID).Select("COALESCE(MAX(position), 0)").Scan(&maxPosition).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check max position")
		}
		if seriesPost.Position > maxPosition+1 {
			seriesPost.Position = maxPosition + 1
		}

		if err := tx.Create(seriesPost).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create series post")
		}

		if err := tx.Model(series).Update("total_posts", series.TotalPosts+1).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update total posts")
		}

		if err := SyncSeriesAnalytics(ctx, rclient, tx, seriesPost.SeriesID, 1, 0); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	rclient.Del(ctx, "series:posts:"+seriesPost.SeriesID.String())
	rclient.Set(ctx, "series:total_posts:"+seriesPost.SeriesID.String(), Series.TotalPosts+1, 24*time.Hour)

	return nil
}

// GetSeriesPost retrieves a series post
func GetSeriesPost(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, condition string, args []interface{}, preload ...string) (*SeriesPost, error) {
	cacheKey := ""
	if condition == "series_id = ? AND post_id = ?" {
		cacheKey = fmt.Sprintf("series_post:%s:%s", args[0].(uuid.UUID).String(), args[1].(uuid.UUID).String())
	} else if condition == "id = ?" {
		cacheKey = "series_post:id:" + args[0].(uuid.UUID).String()
	}
	if cacheKey != "" {
		if cached, err := redisClient.Get(ctx, cacheKey).Result(); err == nil {
			var seriesPost SeriesPost
			if json.Unmarshal([]byte(cached), &seriesPost) == nil {
				return &seriesPost, nil
			}
		}
	}

	query := gormDB.WithContext(ctx).Where(condition, args...)
	for _, p := range preload {
		query = query.Preload(p)
	}

	var seriesPost SeriesPost
	if err := query.First(&seriesPost).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Series post not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch series post")
	}

	if cacheKey != "" {
		data, _ := json.Marshal(seriesPost)
		redisClient.Set(ctx, cacheKey, data, 1*time.Hour)
	}

	return &seriesPost, nil
}
