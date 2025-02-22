package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Posts struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Title            string     `gorm:"size:200;not null;index" json:"title" validate:"required,min=10,max=200"`
	Slug             string     `gorm:"size:220;not null;uniqueIndex" json:"slug" validate:"required,slug,max=220"`
	Content          string     `gorm:"type:text;not null" json:"content" validate:"required,min=100"`
	Excerpt          string     `gorm:"size:300" json:"excerpt" validate:"max=300"`
	FeaturedImageUrl string     `gorm:"size:500" json:"featured_image_url" validate:"omitempty,url,max=500"`
	Published        bool       `gorm:"default:false" json:"published"`
	PublishedAt      *time.Time `json:"published_at"`
	Status           string     `gorm:"type:varchar(20);default:'draft'" json:"status" validate:"required,oneof=draft published unpublished public private"`
	ContentFormat    string     `gorm:"size:20;default:markdown" json:"content_format" validate:"oneof=markdown html"`
	CanonicalURL     string     `gorm:"size:500" json:"canonical_url" validate:"omitempty,url,max=500"`

	// SEO & Social Metadata
	MetaTitle          string `gorm:"size:200" json:"meta_title" validate:"max=200"`
	MetaDescription    string `gorm:"size:300" json:"meta_description" validate:"max=300"`
	SEOKeywords        string `gorm:"size:255" json:"seo_keywords" validate:"omitempty,max=255"`
	OGTitle            string `gorm:"size:200" json:"og_title" validate:"max=200"`
	OGDescription      string `gorm:"size:300" json:"og_description" validate:"max=300"`
	OGImageURL         string `gorm:"size:500" json:"og_image_url" validate:"omitempty,url,max=500"`
	TwitterTitle       string `gorm:"size:200" json:"twitter_title" validate:"max=200"`
	TwitterDescription string `gorm:"size:300" json:"twitter_description" validate:"max=300"`
	TwitterImageURL    string `gorm:"size:500" json:"twitter_image_url" validate:"omitempty,url,max=500"`

	// Collaboration & Review system
	AuthorID       uuid.UUID  `gorm:"type:uuid;not null" json:"author_id"`
	SeriesID       *uuid.UUID `gorm:"type:uuid" json:"series_id"`
	EditedAt       *time.Time `json:"edited_at"`
	LastEditedByID *uuid.UUID `gorm:"type:uuid" json:"last_edited_by_id"`
	NeedsReview    bool       `gorm:"default:false" json:"needs_review"`
	ReviewedByID   *uuid.UUID `gorm:"type:uuid" json:"reviewed_by_id"`
	ReviewedAt     *time.Time `json:"reviewed_at"`

	Author    User       `gorm:"foreignKey:AuthorID" json:"author" validate:"-"`
	Series    Series     `gorm:"foreignKey:SeriesID" json:"series" validate:"-"`
	Tags      []Tag      `gorm:"many2many:post_tags;" json:"tags" validate:"max=4,dive"`
	Comments  []Comment  `gorm:"foreignKey:PostID" json:"comments" validate:"-"`
	Reactions []Reaction `gorm:"foreignKey:ReactableID" json:"reactions" validate:"-"`
	Bookmarks []Bookmark `gorm:"foreignKey:PostID" json:"bookmarks" validate:"-"`
	// SocialPreview SocialMediaPreview `gorm:"embedded;embeddedPrefix:social_preview_" json:"social_preview"`
	Mentions      []User         `gorm:"many2many:post_mentions;" json:"mentions" validate:"valid_mentions,-"`
	CoAuthors     []User         `gorm:"many2many:post_co_authors;" json:"co_authors" validate:"valid_mentions,max=3,dive"`
	PostAnalytics *PostAnalytics `gorm:"foreignKey:PostID" json:"analytics" validate:"-"`

	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// PostAnalytics represents analytics data for a post
type PostAnalytics struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	PostID    uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex" json:"post_id" validate:"required"`
	SeriesID  *uuid.UUID `gorm:"type:uuid" json:"series_id" validate:"omitempty"`
	IpAddress string     `gorm:"size:45" json:"ip_address" validate:"omitempty,ipv4"`

	CommentsCount      int `gorm:"default:0" json:"comments_count"`
	LikesCount         int `gorm:"default:0" json:"likes_count"`
	BookmarksCount     int `gorm:"default:0" json:"bookmarks_count"`
	CompleteCount      int `gorm:"default:0" json:"complete_count"`
	ReadingTimeMinutes int `gorm:"default:1" json:"reading_time_minutes" validate:"min=1"`
	ViewsCount         int `gorm:"default:0" json:"views_count"`
	ShareCount         int `gorm:"default:0" json:"share_count"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// Series represents a collection of related posts
type Series struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Title         string         `gorm:"size:120;not null;uniqueIndex" json:"title" validate:"required,min=5,max=120"`
	Slug          string         `gorm:"size:140;not null;uniqueIndex" json:"slug" validate:"required,slug,max=140"`
	Description   string         `gorm:"type:text;not null" json:"description" validate:"required,min=50,max=1000"`
	CoverImageURL string         `gorm:"size:500" json:"cover_image_url" validate:"omitempty,url,max=500,image_url"`
	AuthorID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"author_id" validate:"required"`
	IsPublished   bool           `gorm:"default:false" json:"is_published"`
	TotalPosts    int            `gorm:"default:0" json:"total_posts" validate:"min=0"`
	CreatedAt     time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Author      User            `gorm:"foreignKey:AuthorID" json:"author" validate:"-"`
	SeriesPosts []SeriesPost    `gorm:"foreignKey:SeriesID" json:"series_posts" validate:"-"`
	Analytics   SeriesAnalytics `gorm:"foreignKey:SeriesID" json:"analytics" validate:"-"`
}

// SeriesPost represents a post in a series with ordering
type SeriesPost struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	SeriesID  uuid.UUID `gorm:"type:uuid;not null;index" json:"series_id" validate:"required"`
	PostID    uuid.UUID `gorm:"type:uuid;not null;index" json:"post_id" validate:"required"`
	Position  int       `gorm:"not null;index:idx_series_position" json:"position" validate:"min=1"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`

	// Relationships
	Post   Posts  `gorm:"foreignKey:PostID" json:"post" validate:"-"`
	Series Series `gorm:"foreignKey:SeriesID" json:"series" validate:"-"`
}

// SeriesAnalytics tracks performance metrics for the series
type SeriesAnalytics struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	SeriesID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"series_id" validate:"required"`

	TotalViews      int     `gorm:"default:0" json:"total_views" validate:"min=0"`
	TotalReactions  int     `gorm:"default:0" json:"total_reactions" validate:"min=0"`
	AverageReadTime float64 `gorm:"default:0.0" json:"average_read_time" validate:"min=0.0"`
	CompletionRate  float64 `gorm:"default:0.0" json:"completion_rate" validate:"min=0.0,max=1.0"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}
