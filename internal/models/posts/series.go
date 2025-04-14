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
