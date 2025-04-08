package models

import (
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
	"gorm.io/gorm"
)

type Tag struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name             string    `gorm:"size:30;not null;uniqueIndex:idx_tag_name" json:"name" validate:"required,min=2,max=30,alphanumunicode"`
	Slug             string    `gorm:"size:35;not null;uniqueIndex:idx_tag_slug" json:"slug" validate:"required,max=35,customSlug"`
	Description      string    `gorm:"size:200" json:"description" validate:"omitempty,max=200"`
	ShortDescription string    `gorm:"size:100" json:"short_description" validate:"omitempty,max=100"`
	IconURL          string    `gorm:"size:500" json:"icon_url" validate:"omitempty,url,max=500"`
	BackgroundURL    string    `gorm:"size:500" json:"background_url" validate:"omitempty,url,max=500"`
	TextColor        string    `gorm:"size:7" json:"text_color" validate:"omitempty,hexcolor,max=7"`
	BackgroundColor  string    `gorm:"size:7" json:"background_color" validate:"omitempty,hexcolor,max=7"`
	IsApproved       bool      `gorm:"default:false;index" json:"is_approved"`
	IsFeatured       bool      `gorm:"default:false;index" json:"is_featured"`
	IsModerated      bool      `gorm:"default:false;index" json:"is_moderated"`
	PostsCount       int       `gorm:"default:0" json:"posts_count" validate:"min=0"`
	FollowersCount   int       `gorm:"default:0" json:"followers_count" validate:"min=0"`
	Rules            string    `gorm:"size:500" json:"rules" validate:"omitempty,max=500"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Posts      []Posts       `gorm:"many2many:post_tags;" json:"posts" validate:"-"`
	Followers  []user.User   `gorm:"many2many:tag_followers;" json:"followers" validate:"-"`
	Moderators []user.User   `gorm:"many2many:tag_moderators;" json:"moderators" validate:"-"`
	Analytics  *TagAnalytics `gorm:"foreignKey:TagID" json:"analytics" validate:"-"`
}

// TagAnalytics represents analytics data for a tag
type TagAnalytics struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TagID            uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_tag_analytics_tag_id" json:"tag_id" validate:"required"`
	DailyViews       int       `gorm:"default:0" json:"daily_views" validate:"min=0"`
	WeeklyViews      int       `gorm:"default:0" json:"weekly_views" validate:"min=0"`
	MonthlyViews     int       `gorm:"default:0" json:"monthly_views" validate:"min=0"`
	DailyFollowers   int       `gorm:"default:0" json:"daily_followers" validate:"min=0"`
	WeeklyFollowers  int       `gorm:"default:0" json:"weekly_followers" validate:"min=0"`
	MonthlyFollowers int       `gorm:"default:0" json:"monthly_followers" validate:"min=0"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Tag Tag `gorm:"foreignKey:TagID" json:"tag" validate:"-"`
}

// TagFollower represents users following a tag
type TagFollower struct {
	TagID     uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"tag_id" validate:"required"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"user_id" validate:"required"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	Tag  Tag       `gorm:"foreignKey:TagID" json:"tag" validate:"-"`
	User user.User `gorm:"foreignKey:UserID" json:"user" validate:"-"`
}

// TagModerator represents moderators for a tag
type TagModerator struct {
	TagID     uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"tag_id" validate:"required"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"user_id" validate:"required"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	Tag  Tag       `gorm:"foreignKey:TagID" json:"tag" validate:"-"`
	User user.User `gorm:"foreignKey:UserID" json:"user" validate:"-"`
}
