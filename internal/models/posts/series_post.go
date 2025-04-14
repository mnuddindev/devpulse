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
func GetSeriesPost(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, condition string, args []interface{}, preload ...string) (*SeriesPost, error) {
	cacheKey := ""
	if condition == "series_id = ? AND post_id = ?" {
		cacheKey = fmt.Sprintf("series_post:%s:%s", args[0].(uuid.UUID).String(), args[1].(uuid.UUID).String())
	} else if condition == "id = ?" {
		cacheKey = "series_post:id:" + args[0].(uuid.UUID).String()
	}
	if cacheKey != "" {
		if cached, err := rclient.Get(ctx, cacheKey).Result(); err == nil {
			var seriesPost SeriesPost
			if json.Unmarshal([]byte(cached), &seriesPost) == nil {
				return &seriesPost, nil
			}
		}
	}

	query := db.WithContext(ctx).Where(condition, args...)
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
		rclient.Set(ctx, cacheKey, data, 1*time.Hour)
	}

	return &seriesPost, nil
}

// UpdateSeriesPost modifies a series post
func UpdateSeriesPost(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, seriesPostID uuid.UUID, options ...SeriesPostOption) (*SeriesPost, error) {
	tx := db.WithContext(ctx).Begin()
	seriesPost, err := GetSeriesPost(ctx, rclient, db, "id = ?", []interface{}{seriesPostID})
	if err != nil {
		return nil, err
	}

	originalSeriesID := seriesPost.SeriesID
	for _, opt := range options {
		opt(seriesPost)
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		if seriesPost.PostID != seriesPost.PostID && seriesPost.PostID != uuid.Nil {
			_, err := GetPostsBy(ctx, rclient, tx, "id = ?", []interface{}{seriesPost.PostID})
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return utils.NewError(utils.ErrNotFound.Code, "Post not found")
				}
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch post")
			}
		}

		if seriesPost.SeriesID != originalSeriesID || seriesPost.PostID != seriesPost.PostID {
			var existing SeriesPost
			if err := tx.Where("series_id = ? AND post_id = ?", seriesPost.SeriesID, seriesPost.PostID).First(&existing).Error; err == nil {
				return utils.NewError(utils.ErrBadRequest.Code, "Post already in series")
			} else if err != gorm.ErrRecordNotFound {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check series post")
			}
		}

		var maxPosition int
		if err := tx.Model(&SeriesPost{}).Where("series_id = ?", seriesPost.SeriesID).Select("COALESCE(MAX(position), 0)").Scan(&maxPosition).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check max position")
		}
		if seriesPost.Position > maxPosition {
			seriesPost.Position = maxPosition
		}

		if err := tx.Save(seriesPost).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update series post")
		}

		if seriesPost.SeriesID != originalSeriesID {
			if err := tx.Model(&Series{ID: originalSeriesID}).Update("total_posts", gorm.Expr("total_posts - 1")).Error; err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update old series total posts")
			}
			if err := tx.Model(&Series{ID: seriesPost.SeriesID}).Update("total_posts", gorm.Expr("total_posts + 1")).Error; err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update new series total posts")
			}
			if err := SyncSeriesAnalytics(ctx, rclient, tx, originalSeriesID, -1, 0); err != nil {
				return err
			}
			if err := SyncSeriesAnalytics(ctx, rclient, tx, seriesPost.SeriesID, 1, 0); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()

	rclient.Del(ctx,
		"series:posts:"+seriesPost.SeriesID.String(),
		"series:posts:"+originalSeriesID.String(),
	)
	if seriesPost.SeriesID != originalSeriesID {
		rclient.Set(ctx, "series:total_posts:"+seriesPost.SeriesID.String(), Series.TotalPosts+1, 24*time.Hour)
		rclient.Set(ctx, "series:total_posts:"+originalSeriesID.String(), Series.TotalPosts-1, 24*time.Hour)
	}

	return seriesPost, nil
}

// DeleteSeriesPost removes a post from a series
func DeleteSeriesPost(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, seriesID, postID uuid.UUID) error {
	tx := db.WithContext(ctx).Begin()
	series, err := GetSeries(ctx, rclient, db, "id = ?", []interface{}{seriesID})
	if err != nil {
		return err
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		// Delete series post
		result := tx.Where("series_id = ? AND post_id = ?", seriesID, postID).Delete(&SeriesPost{})
		if result.Error != nil {
			return utils.WrapError(result.Error, utils.ErrInternalServerError.Code, "Failed to delete series post")
		}
		if result.RowsAffected == 0 {
			return utils.NewError(utils.ErrNotFound.Code, "Series post not found")
		}

		// Update TotalPosts
		if series.TotalPosts > 0 {
			series.TotalPosts--
			if err := tx.Model(series).Update("total_posts", series.TotalPosts).Error; err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update total posts")
			}
		}

		// Sync analytics
		if err := SyncSeriesAnalytics(ctx, rclient, tx, seriesID, -1, 0); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	// Update caches
	seriesData, _ := json.Marshal(series)
	rclient.Set(ctx, "series:"+series.ID.String(), seriesData, 24*time.Hour)
	rclient.Set(ctx, "series:slug:"+series.Slug, seriesData, 24*time.Hour)
	rclient.Set(ctx, "series:total_posts:"+seriesID.String(), series.TotalPosts, 24*time.Hour)
	rclient.Del(ctx, "series:posts:"+seriesID.String())

	return nil
}

// ListSeriesPosts retrieves all posts in a series
func ListSeriesPosts(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, seriesID uuid.UUID, page, limit int) ([]SeriesPost, int64, error) {
	if page < 1 || limit < 1 {
		return nil, 0, utils.NewError(utils.ErrBadRequest.Code, "Invalid page or limit")
	}

	cacheKey := fmt.Sprintf("series:posts:%s:page:%d:limit:%d", seriesID.String(), page, limit)
	if cached, err := rclient.Get(ctx, cacheKey).Result(); err == nil {
		var result struct {
			Posts []SeriesPost
			Total int64
		}
		if json.Unmarshal([]byte(cached), &result) == nil {
			return result.Posts, result.Total, nil
		}
	}

	var total int64
	if err := db.WithContext(ctx).Model(&SeriesPost{}).Where("series_id = ?", seriesID).Count(&total).Error; err != nil {
		return nil, 0, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to count series posts")
	}

	var seriesPosts []SeriesPost
	offset := (page - 1) * limit
	if err := db.WithContext(ctx).
		Where("series_id = ?", seriesID).
		Offset(offset).
		Limit(limit).
		Order("position ASC").
		Preload("Post").
		Find(&seriesPosts).Error; err != nil {
		return nil, 0, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch series posts")
	}

	// Cache result
	result := struct {
		Posts []SeriesPost
		Total int64
	}{seriesPosts, total}
	data, _ := json.Marshal(result)
	rclient.Set(ctx, cacheKey, data, 1*time.Hour)

	return seriesPosts, total, nil
}

// ReorderSeriesPosts updates positions for multiple series posts
func ReorderSeriesPosts(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, seriesID uuid.UUID, positions map[uuid.UUID]int) error {
	if len(positions) == 0 {
		return utils.NewError(utils.ErrBadRequest.Code, "No positions provided")
	}

	tx := db.WithContext(ctx).Begin()
	_, err := GetSeries(ctx, rclient, db, "id = ?", []interface{}{seriesID})
	if err != nil {
		return err
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		var seriesPosts []SeriesPost
		if err := tx.Where("series_id = ? AND post_id IN ?", seriesID, getKeys(positions)).Find(&seriesPosts).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch series posts")
		}

		usedPositions := make(map[int]bool)
		for postID, pos := range positions {
			if pos < 1 {
				return utils.NewError(utils.ErrBadRequest.Code, fmt.Sprintf("Invalid position %d for post %s", pos, postID.String()))
			}
			if usedPositions[pos] {
				return utils.NewError(utils.ErrBadRequest.Code, fmt.Sprintf("Duplicate position %d", pos))
			}
			usedPositions[pos] = true
		}

		for _, sp := range seriesPosts {
			if newPos, exists := positions[sp.PostID]; exists && newPos != sp.Position {
				if err := tx.Model(&SeriesPost{}).
					Where("id = ?", sp.ID).
					Update("position", newPos).Error; err != nil {
					return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update series post position")
				}
			}
		}

		return nil
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	rclient.Del(ctx, "series:posts:"+seriesID.String())

	return nil
}

// getKeys extracts keys from a map
func getKeys(m map[uuid.UUID]int) []uuid.UUID {
	keys := make([]uuid.UUID, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
