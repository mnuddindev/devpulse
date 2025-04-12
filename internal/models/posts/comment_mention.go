package models

import (
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
)

type CommentMention struct {
	CommentID uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"comment_id" validate:"required"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"user_id" validate:"required"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	Comment Comment   `gorm:"foreignKey:CommentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"comment" validate:"-"`
	User    user.User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"user" validate:"-"`
}
