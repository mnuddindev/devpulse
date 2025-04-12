package models

import (
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
	"gorm.io/gorm"
)

type Collection struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index:idx_collection_user" json:"user_id" validate:"required"`
	Name        string    `gorm:"size:100;not null" json:"name" validate:"required,min=3,max=100"`
	Description string    `gorm:"size:200" json:"description" validate:"omitempty,max=200"`
	IsPrivate   bool      `gorm:"default:false;index" json:"is_private"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User      user.User  `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"user" validate:"-"`
	Bookmarks []Bookmark `gorm:"foreignKey:CollectionID" json:"bookmarks" validate:"-"`
}
