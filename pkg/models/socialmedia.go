package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SocialMediaPreview represents metadata for generating social media previews
type SocialMediaPreview struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	PostID         uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"post_id" validate:"required"`
	Title          string         `gorm:"size:120;not null" json:"title" validate:"required,min=10,max=120"`
	Description    string         `gorm:"size:200;not null" json:"description" validate:"required,min=20,max=200"`
	ImageURL       string         `gorm:"size:500;not null" json:"image_url" validate:"required,url,max=500"`
	AuthorName     string         `gorm:"size:100;not null" json:"author_name" validate:"required,min=3,max=100"`
	AuthorHandle   string         `gorm:"size:50;not null" json:"author_handle" validate:"required,min=3,max=50"`
	AuthorImageURL string         `gorm:"size:500" json:"author_image_url" validate:"omitempty,url,max=500"`
	SiteName       string         `gorm:"size:50;not null" json:"site_name" validate:"required,min=3,max=50"`
	SiteURL        string         `gorm:"size:500;not null" json:"site_url" validate:"required,url,max=500"`
	ThemeColor     string         `gorm:"size:7;not null" json:"theme_color" validate:"required,hexcolor|max=7"`
	CreatedAt      time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Post *Posts `gorm:"foreignKey:PostID" json:"post" validate:"-"`
}
