package models

import (
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
	"gorm.io/gorm"
)

type Reaction struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index:idx_reaction_user" json:"user_id" validate:"required"`
	PostID        uuid.UUID `gorm:"type:uuid;not null;index:idx_reaction_post" json:"post_id" validate:"required"`
	ReactableID   uuid.UUID `gorm:"type:uuid;not null" json:"reactable_id" validate:"required"`
	ReactableType string    `gorm:"size:50;not null" json:"reactable_type" validate:"required,oneof=post comment"`
	Type          string    `gorm:"size:20;not null;index:idx_reaction_type" json:"type" validate:"required,oneof=love unicorn explosion support fire"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User    user.User `gorm:"foreignKey:UserID" json:"user" validate:"-"`
	Post    Posts     `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"post,omitempty" validate:"-"`
	Comment Comment   `gorm:"foreignKey:ReactableID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"comment,omitempty" validate:"-"`
}

type ReadingListEntry struct {
	ReactionID uuid.UUID `gorm:"type:uuid;primaryKey" json:"reaction_id" validate:"required"`
	Notes      string    `gorm:"type:text" json:"notes" validate:"omitempty,max=500"`
	IsPrivate  bool      `gorm:"default:false;index" json:"is_private"`
	Tags       []string  `gorm:"type:text[];index:idx_reading_tags,gin" json:"tags" validate:"max=5,dive,max=20,alphanumunicode"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Reaction Reaction `gorm:"foreignKey:ReactionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"reaction" validate:"-"`
}
