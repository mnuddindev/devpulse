package models

import (
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
	"gorm.io/gorm"
)

type Comment struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Content         string     `gorm:"type:text;not null" json:"content" validate:"required,min=2,max=1000"`
	PostID          uuid.UUID  `gorm:"type:uuid;not null;index:idx_comment_post" json:"post_id" validate:"required"`
	AuthorID        uuid.UUID  `gorm:"type:uuid;not null;index:idx_comment_author" json:"author_id" validate:"required"`
	ParentCommentID *uuid.UUID `gorm:"type:uuid;index:idx_comment_parent" json:"parent_comment_id" validate:"omitempty"`
	Depth           int        `gorm:"default:0;index" json:"depth" validate:"min=0,max=10"`
	UpvotesCount    int        `gorm:"default:0" json:"upvotes_count" validate:"min=0"`
	DownvotesCount  int        `gorm:"default:0" json:"downvotes_count" validate:"min=0"`
	Edited          bool       `gorm:"default:false;index" json:"edited"`
	Pinned          bool       `gorm:"default:false;index" json:"pinned"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Author        user.User     `gorm:"foreignKey:AuthorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"author" validate:"-"`
	Post          Posts         `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"post" validate:"-"`
	ParentComment *Comment      `gorm:"foreignKey:ParentCommentID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"parent_comment" validate:"-"`
	Replies       []Comment     `gorm:"foreignKey:ParentCommentID" json:"replies" validate:"-"`
	Mentions      []user.User   `gorm:"many2many:comment_mentions;" json:"mentions" validate:"max=5,dive"`
	Reactions     []Reaction    `gorm:"foreignKey:ReactableID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"reactions" validate:"-"`
	Flags         []CommentFlag `gorm:"foreignKey:CommentID" json:"flags" validate:"-"`
}
