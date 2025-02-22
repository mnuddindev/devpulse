package models

import (
	"time"

	"github.com/google/uuid"
)

// Reaction represents a user reaction to a post or comment
type Reaction struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id" validate:"required"`
	PostID        uuid.UUID `gorm:"type:uuid;not null;index" json:"post_id" validate:"required"`
	ReactableID   uuid.UUID `gorm:"type:uuid;not null;index:idx_reactable" json:"reactable_id" validate:"required"`
	ReactableType string    `gorm:"size:50;not null;index:idx_reactable" json:"reactable_type" validate:"required,oneof=post comment"`
	Type          string    `gorm:"size:20;not null;index" json:"type" validate:"required,oneof=love unicorn explosion support fire"`
	CreatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`

	// Relationships
	User    User    `gorm:"foreignKey:UserID" json:"user" validate:"-"`
	Post    Posts   `gorm:"foreignKey:ReactableID" json:"post,omitempty" validate:"-"`
	Comment Comment `gorm:"foreignKey:ReactableID" json:"comment,omitempty" validate:"-"`
}

// ReadingListEntry represents a post saved to a user's reading list (specialized reaction)
type ReadingListEntry struct {
	ReactionID uuid.UUID `gorm:"type:uuid;primaryKey" json:"reaction_id"`
	Notes      string    `gorm:"type:text" json:"notes" validate:"max=500"`
	IsPrivate  bool      `gorm:"default:false" json:"is_private"`
	Tags       []string  `gorm:"type:text[]" json:"tags" validate:"max=5,dive,max=20"`

	Reaction Reaction `gorm:"foreignKey:ReactionID" json:"reaction" validate:"-"`
}
