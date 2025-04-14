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

type Series struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Title         string    `gorm:"size:120;not null;uniqueIndex:idx_series_title" json:"title" validate:"required,min=5,max=120"`
	Slug          string    `gorm:"size:140;not null;uniqueIndex:idx_series_slug" json:"slug" validate:"required,max=140,customSlug"`
	Description   string    `gorm:"type:text;not null" json:"description" validate:"required,min=50,max=1000"`
	CoverImageURL string    `gorm:"size:500" json:"cover_image_url" validate:"omitempty,url,max=500"`
	AuthorID      uuid.UUID `gorm:"type:uuid;not null;index:idx_series_author" json:"author_id" validate:"required"`
	IsPublished   bool      `gorm:"default:false;index" json:"is_published"`
	TotalPosts    int       `gorm:"default:0" json:"total_posts" validate:"min=0"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Author      user.User       `gorm:"foreignKey:AuthorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"author" validate:"-"`
	SeriesPosts []SeriesPost    `gorm:"foreignKey:SeriesID" json:"series_posts" validate:"-"`
	Analytics   SeriesAnalytics `gorm:"foreignKey:SeriesID" json:"analytics" validate:"-"`
}

// SeriesOption defines a function type that takes a pointer to a Tag and modifies it.
type SeriesOption func(*Series)

// CreateSeries initializes a new series with analytics
func CreateSeries(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, series *Series) error {
	if series.ID == uuid.Nil {
		series.ID = uuid.New()
	}
	series.Title = strings.TrimSpace(series.Title)
	series.Slug = strings.ToLower(strings.TrimSpace(series.Slug))
	series.Description = strings.TrimSpace(series.Description)
	series.CoverImageURL = strings.TrimSpace(series.CoverImageURL)

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingByTitle Series
		if err := tx.Where("title = ?", series.Title).First(&existingByTitle).Error; err == nil {
			return utils.NewError(utils.ErrBadRequest.Code, "Series title already exists")
		} else if err != gorm.ErrRecordNotFound {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check series title")
		}

		_, err := user.GetUserBy(ctx, rclient, tx, "id = ?", []interface{}{series.AuthorID})
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return utils.NewError(utils.ErrNotFound.Code, "Author not found")
			}
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch author")
		}

		if err := tx.Create(series).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create series")
		}

		// analytics := &SeriesAnalytics{SeriesID: series.ID}
		// if err := CreateSeriesAnalytics(ctx, redisClient, tx, analytics); err != nil {
		// 	return err
		// }

		return nil
	})

	if err != nil {
		return err
	}

	seriesData, _ := json.Marshal(series)
	rclient.Set(ctx, "series:"+series.ID.String(), seriesData, 24*time.Hour)
	rclient.Set(ctx, "series:slug:"+series.Slug, seriesData, 24*time.Hour)

	return nil
}

// GetSeries retrieves a series by condition
func GetSeries(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, condition string, args []interface{}, preload ...string) (*Series, error) {
	cacheKey := ""
	switch condition {
	case "id = ?":
		cacheKey = "series:" + args[0].(uuid.UUID).String()
	case "slug = ?":
		cacheKey = "series:slug:" + args[0].(string)
	}
	if cacheKey != "" {
		if cached, err := rclient.Get(ctx, cacheKey).Result(); err == nil {
			var series Series
			if json.Unmarshal([]byte(cached), &series) == nil {
				return &series, nil
			}
		}
	}

	query := db.WithContext(ctx).Where(condition, args...)
	for _, p := range preload {
		query = query.Preload(p)
	}

	var series Series
	if err := query.First(&series).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Series not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch series")
	}

	if cacheKey != "" {
		seriesData, _ := json.Marshal(series)
		rclient.Set(ctx, cacheKey, seriesData, 24*time.Hour)
	}

	return &series, nil
}

// UpdateSeries modifies a series using options
func UpdateSeries(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, seriesID uuid.UUID, options ...SeriesOption) (*Series, error) {
	tx := db.WithContext(ctx).Begin()
	series, err := GetSeries(ctx, rclient, db, "id = ?", []interface{}{seriesID}, "")
	if err != nil {
		return nil, err
	}

	originalSlug := series.Slug
	for _, opt := range options {
		opt(series)
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		if series.Title != "" && series.Title != series.Title {
			var existingByTitle Series
			if err := tx.Where("title = ? AND id != ?", series.Title, series.ID).First(&existingByTitle).Error; err == nil {
				return utils.NewError(utils.ErrBadRequest.Code, "Series title already exists")
			} else if err != gorm.ErrRecordNotFound {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check series title")
			}
		}

		if series.Slug != originalSlug && series.Slug != "" {
			var existingBySlug Series
			if err := tx.Where("slug = ? AND id != ?", series.Slug, series.ID).First(&existingBySlug).Error; err == nil {
				return utils.NewError(utils.ErrBadRequest.Code, "Series slug already exists")
			} else if err != gorm.ErrRecordNotFound {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check series slug")
			}
		}

		if series.AuthorID != uuid.Nil && series.AuthorID != series.AuthorID {
			_, err := user.GetUserBy(ctx, rclient, tx, "id = ?", []interface{}{series.AuthorID})
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return utils.NewError(utils.ErrNotFound.Code, "Author not found")
				}
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch author")
			}
		}

		if err := tx.Save(series).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update series")
		}

		return nil
	})
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()

	seriesData, _ := json.Marshal(series)
	rclient.Set(ctx, "series:"+series.ID.String(), seriesData, 24*time.Hour)
	rclient.Set(ctx, "series:slug:"+series.Slug, seriesData, 24*time.Hour)
	if series.Slug != originalSlug {
		rclient.Del(ctx, "series:slug:"+originalSlug)
	}
	rclient.Set(ctx, "series:total_posts:"+series.ID.String(), series.TotalPosts, 24*time.Hour)

	return series, nil
}
