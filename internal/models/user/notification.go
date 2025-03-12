package models

import (
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Type      string    `gorm:"size:50;not null" json:"type"`
	Message   string    `gorm:"size:255;not null" json:"message"`
	IsRead    bool      `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type NotificationPreferences struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;unique" json:"user_id"`
	EmailOnLikes     bool      `gorm:"default:false" json:"email_on_likes"`
	EmailOnComments  bool      `gorm:"default:false" json:"email_on_comments"`
	EmailOnMentions  bool      `gorm:"default:false" json:"email_on_mentions"`
	EmailOnFollowers bool      `gorm:"default:false" json:"email_on_followers"`
	EmailOnBadge     bool      `gorm:"default:false" json:"email_on_badge"`
	EmailOnUnread    bool      `gorm:"default:false" json:"email_on_unread"`
	EmailOnNewPosts  bool      `gorm:"default:false" json:"email_on_new_posts"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
