package models

import (
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
)

type CommentFlag struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	CommentID uuid.UUID `gorm:"type:uuid;not null;index:idx_flag_comment" json:"comment_id" validate:"required"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_flag_user" json:"user_id" validate:"required"`
	Reason    string    `gorm:"size:100;not null" json:"reason" validate:"required,oneof=spam inappropriate harassment other"`
	Notes     string    `gorm:"size:500" json:"notes" validate:"omitempty,max=500"`
	Resolved  bool      `gorm:"default:false;index" json:"resolved"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	User    user.User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"user" validate:"-"`
	Comment Comment   `gorm:"foreignKey:CommentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"comment" validate:"-"`
}
