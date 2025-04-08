package models

import (
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
)

type TagModerator struct {
	TagID     uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"tag_id" validate:"required"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"user_id" validate:"required"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	Tag  Tag       `gorm:"foreignKey:TagID" json:"tag" validate:"-"`
	User user.User `gorm:"foreignKey:UserID" json:"user" validate:"-"`
}
