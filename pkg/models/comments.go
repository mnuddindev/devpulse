package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Comment represents a comment on a post or another comment
type Comment struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Content         string     `gorm:"type:text;not null" json:"content" validate:"required,min=2,max=1000"`
	PostID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"post_id" validate:"required"`
	AuthorID        uuid.UUID  `gorm:"type:uuid;not null" json:"author_id" validate:"required"`
	ParentCommentID *uuid.UUID `gorm:"type:uuid;index" json:"parent_comment_id" validate:"omitempty"`
	Depth           int        `gorm:"default:0" json:"depth" validate:"min=0,max=10"`
	UpvotesCount    int        `gorm:"default:0" json:"upvotes_count" validate:"min=0"`
	DownvotesCount  int        `gorm:"default:0" json:"downvotes_count" validate:"min=0"`
	Edited          bool       `gorm:"default:false" json:"edited"`
	Pinned          bool       `gorm:"default:false" json:"pinned"`

	Author        User          `gorm:"foreignKey:AuthorID" json:"author" validate:"-"`
	Post          Posts         `gorm:"foreignKey:PostID" json:"post" validate:"-"`
	ParentComment *Comment      `gorm:"foreignKey:ParentCommentID" json:"parent_comment" validate:"-"`
	Replies       []Comment     `gorm:"foreignKey:ParentCommentID" json:"replies" validate:"-"`
	Mentions      []User        `gorm:"many2many:comment_mentions;" json:"mentions" validate:"valid_mentions,max=5,dive"`
	Reactions     []Reaction    `gorm:"foreignKey:ReactableID;" json:"reactions" validate:"-"`
	Flags         []CommentFlag `gorm:"foreignKey:CommentID" json:"flags" validate:"-"`

	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// CommentFlag represents flags/reports on a comment (e.g., spam, inappropriate, etc.)
type CommentFlag struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	CommentID uuid.UUID `gorm:"type:uuid;not null;index" json:"comment_id" validate:"required"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id" validate:"required"`
	Reason    string    `gorm:"size:100;not null" json:"reason" validate:"required,oneof=spam inappropriate harassment other"`
	Notes     string    `gorm:"size:500" json:"notes" validate:"max=500"`
	Resolved  bool      `gorm:"default:false" json:"resolved"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`

	User    User    `gorm:"foreignKey:UserID" json:"user" validate:"-"`
	Comment Comment `gorm:"foreignKey:CommentID" json:"comment" validate:"-"`
}

// CommentMention represents mentions of users in comments
type CommentMention struct {
	CommentID uuid.UUID `gorm:"type:uuid;primaryKey" json:"comment_id"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`

	Comment Comment `gorm:"foreignKey:CommentID" json:"comment" validate:"-"`
	User    User    `gorm:"foreignKey:UserID" json:"user" validate:"-"`
}
