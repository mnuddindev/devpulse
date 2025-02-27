package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tag represents a tag/category for posts
type Tag struct {
	ID               uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name             string         `gorm:"size:30;not null;uniqueIndex" json:"name" validate:"required,min=2,max=30,alphanumunicode"`
	Slug             string         `gorm:"size:35;not null;uniqueIndex" json:"slug" validate:"required,slug,max=35"`
	Deescription     string         `gorm:"size:200" json:"description" validate:"max=200"`
	ShortDescription string         `gorm:"size:100" json:"short_description" validate:"max=100"`
	IconURL          string         `gorm:"size:500" json:"icon_url" validate:"omitempty,url,max=500"`
	BackgroundURL    string         `gorm:"size:500" json:"background_url" validate:"omitempty,url,max=500"`
	TextColor        string         `gorm:"size:7" json:"text_color" validate:"omitempty,hexcolor|max=7"`
	BackgroundColor  string         `gorm:"size:7" json:"background_color" validate:"omitempty,hexcolor|max=7"`
	IsApproved       bool           `gorm:"default:false" json:"is_approved"`  // Whether the tag is approved by moderators
	IsFeatured       bool           `gorm:"default:false" json:"is_featured"`  // Whether the tag is featured
	IsModerated      bool           `gorm:"default:false" json:"is_moderated"` // Whether the tag requires moderation
	PostsCount       int            `gorm:"default:0" json:"posts_count" validate:"min=0"`
	FollowersCount   int            `gorm:"default:0" json:"followers_count" validate:"min=0"`
	Rules            string         `gorm:"size:500" json:"rules" validate:"max=500"` // Community rules for the tag
	CreatedAt        time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Posts      []Posts      `gorm:"many2many:post_tags;" json:"posts" validate:"-"`
	Followers  []User       `gorm:"many2many:tag_followers;" json:"followers" validate:"-"`
	Moderators []User       `gorm:"many2many:tag_moderators;" json:"moderators" validate:"-"`
	Analytics  *TagAnalytics `gorm:"foreignKey:TagID" json:"analytics" validate:"-"`
}

// TagAnalytics represents analytics data for a tag
type TagAnalytics struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	TagID            uuid.UUID `gorm:"type:uuid;not null;index" json:"tag_id" validate:"required"`
	DailyViews       int       `gorm:"default:0" json:"daily_views" validate:"min=0"`
	WeeklyViews      int       `gorm:"default:0" json:"weekly_views" validate:"min=0"`
	MonthlyViews     int       `gorm:"default:0" json:"monthly_views" validate:"min=0"`
	DailyFollowers   int       `gorm:"default:0" json:"daily_followers" validate:"min=0"`
	WeeklyFollowers  int       `gorm:"default:0" json:"weekly_followers" validate:"min=0"`
	MonthlyFollowers int       `gorm:"default:0" json:"monthly_followers" validate:"min=0"`
	CreatedAt        time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`

	Tag Tag `gorm:"foreignKey:TagID" json:"tag" validate:"-"`
}

// TagFollower represents users following a tag
type TagFollower struct {
	TagID     uuid.UUID `gorm:"type:uuid;primaryKey" json:"tag_id"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`

	Tag  *Tag  `gorm:"foreignKey:TagID" json:"tag" validate:"-"`
	User User `gorm:"foreignKey:UserID" json:"user" validate:"-"`
}

// TagModerator represents moderators for a tag
type TagModerator struct {
	TagID     uuid.UUID `gorm:"type:uuid;primaryKey" json:"tag_id"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`

	Tag  *Tag  `gorm:"foreignKey:TagID" json:"tag" validate:"-"`
	User User `gorm:"foreignKey:UserID" json:"user" validate:"-"`
}
