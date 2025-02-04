package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	// Use google uuid to generate a unique id for each user
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	// Unique username for each user
	Username string `gorm:"size:255;not null;unique" json:"username" validate:"required,min=3"`
	// Unique email for each user
	Email string `gorm:"size:100;not null;unique" json:"email" validate:"required,email"`
	// Password hash for each user (never expose this to the client)
	Password string `gorm:"not null;" json:"-" validate:"required,min=6"`
	// First name of the user
	FirstName string `gorm:"size:100;not null;" json:"first_name" validate:"required,min=3"`
	// Last name of the user
	LastName string `gorm:"size:100;not null;" json:"last_name" validate:"required,min=3"`
	// Bio of the user
	Bio string `gorm:"type:text;size:255;" json:"bio" validate:"max=255"`
	// Avatar Url of the user
	AvatarUrl string `gorm:"type:text;size:255;" json:"avatar_url" validate:"url"`
	// Job Title of the user
	JobTitle string `gorm:"size:100;" json:"job_title" validate:"max=100"`
	// Company of the user
	Employer string `gorm:"size:100;" json:"employer" validate:"max=100"`
	// Location of the user
	Location string `gorm:"size:100;" json:"location" validate:"max=100"`
	// GithubURL of the user
	GithubUrl string `gorm:"size:255;" json:"github_url" validate:"url"`
	// Website of the user
	Website string `gorm:"size:255;" json:"website" validate:"url"`
	// Skills of the user
	Skills []string `gorm:"type:text[]" json:"skills" validate:"max=255"`
	// Interests of the user
	Interests []string `gorm:"type:text[]" json:"interests" validate:"max=255"`
	// User account activation
	IsActive bool `gorm:"default:false" json:"is_active"`
	// Admin Role
	IsAdmin bool `gorm:"default:false" json:"is_admin"`
	// Is moderator role
	IsModerator bool `gorm:"default:false" json:"is_moderator"`
	// trusted user or not
	IsTrusted bool `gorm:"default:false" json:"is_trusted"`
	// Email verified or not
	IsEmailVerified bool `gorm:"default:false" json:"is_email_verified"`
	// Last login time
	LastLogin time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"last_login"`
	// number of followers
	FollowersCount int `gorm:"default:0" json:"followers_count"`
	// number of following users
	FollowingCount int `gorm:"default:0" json:"following_count"`
	// number of posts
	PostsCount int `gorm:"default:0" json:"posts_count"`
	// number of comments
	CommentsCount int `gorm:"default:0" json:"comments_count"`
	// number of likes
	LikesCount int `gorm:"default:0" json:"likes_count"`
	// number of bookmarks
	BookmarksCount int `gorm:"default:0" json:"bookmarks_count"`
	// notifications
	Notifications []Notification `gorm:"foreignkey:UserID" json:"notifications"`
	// theme preference of the user
	ThemePreference string    `gorm:"default:light" json:"theme_preference"`
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type Notification struct {
	// Send notification to user if any event occurs
	UserID          uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	EmailOnLikes    bool      `gorm:"default:false" json:"email_on_likes"`
	EmailOnComments bool      `gorm:"default:false" json:"email_on_comments"`
	EmailOnMentions bool      `gorm:"default:false" json:"email_on_mentions"`
	EmailOnFollower bool      `gorm:"default:false" json:"email_on_followers"`
	EmailOnNewPosts bool      `gorm:"default:false" json:"email_on_new_posts"`
}
