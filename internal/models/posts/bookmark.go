package models

import (
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
	"gorm.io/gorm"
)

type Bookmark struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_bookmark_user" json:"user_id" validate:"required"`
	PostID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_bookmark_post" json:"post_id" validate:"required"`
	CollectionID *uuid.UUID `gorm:"type:uuid;index:idx_bookmark_collection" json:"collection_id" validate:"omitempty"`
	Notes        string     `gorm:"type:text" json:"notes" validate:"omitempty,max=500"`
	IsPrivate    bool       `gorm:"default:false;index" json:"is_private"`
	Tags         []string   `gorm:"type:text[];index:idx_bookmark_tags,gin" json:"tags" validate:"max=5,dive,max=20,alphanumunicode"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User       user.User   `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"user" validate:"-"`
	Post       Posts       `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"post" validate:"-"`
	Collection *Collection `gorm:"foreignKey:CollectionID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"collection" validate:"-"`
}
