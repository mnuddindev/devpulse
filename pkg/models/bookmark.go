package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Bookmark represents a user's bookmark of a post
type Bookmark struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id" validate:"required"`
	PostID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"post_id" validate:"required"`
	CollectionID *uuid.UUID     `gorm:"type:uuid;index" json:"collection_id" validate:"omitempty"`
	Notes        string         `gorm:"type:text" json:"notes" validate:"max=500"`
	IsPrivate    bool           `gorm:"default:false" json:"is_private"`
	Tags         []string       `gorm:"type:text[]" json:"tags" validate:"max=5,dive,max=20"`
	CreatedAt    time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User       User        `gorm:"foreignKey:UserID" json:"user" validate:"-"`
	Post       Posts       `gorm:"foreignKey:PostID" json:"post" validate:"-"`
	Collection *Collection `gorm:"foreignKey:CollectionID" json:"collection" validate:"-"`
}

// Collection represents a group of bookmarks
type Collection struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id" validate:"required"`
	Name        string         `gorm:"size:100;not null" json:"name" validate:"required,min=3,max=100"`
	Description string         `gorm:"size:200" json:"description" validate:"max=200"`
	IsPrivate   bool           `gorm:"default:false" json:"is_private"`
	CreatedAt   time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User      User       `gorm:"foreignKey:UserID" json:"user" validate:"-"`
	Bookmarks []Bookmark `gorm:"foreignKey:CollectionID" json:"bookmarks" validate:"-"`
}
