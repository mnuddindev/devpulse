package models

import (
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
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

	// Relationships
	Author      user.User       `gorm:"foreignKey:AuthorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"author" validate:"-"`
	SeriesPosts []SeriesPost    `gorm:"foreignKey:SeriesID" json:"series_posts" validate:"-"`
	Analytics   SeriesAnalytics `gorm:"foreignKey:SeriesID" json:"analytics" validate:"-"`
}

type SeriesPost struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SeriesID uuid.UUID `gorm:"type:uuid;not null;index:idx_series_post_series" json:"series_id" validate:"required"`
	PostID   uuid.UUID `gorm:"type:uuid;not null;index:idx_series_post_post" json:"post_id" validate:"required"`
	Position int       `gorm:"not null;index:idx_series_position" json:"position" validate:"min=1"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Post   Posts  `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"post" validate:"-"`
	Series Series `gorm:"foreignKey:SeriesID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"series" validate:"-"`
}

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
