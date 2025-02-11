package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	// Use google uuid to generate a unique id for each user
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	// Unique username for each user
	Username string `gorm:"size:255;not null;unique" json:"username" validate:"required,min=3"`
	// Unique email for each user
	Email string `gorm:"size:100;not null;unique" json:"email" validate:"required,email"`
	// Password hash for each user (never expose this to the client)
	Password string `gorm:"not null;" json:"password" validate:"required,min=6"`
	// First name of the user
	FirstName string `gorm:"size:100;not null;" json:"first_name" validate:"required,min=3"`
	// Last name of the user
	LastName string `gorm:"size:100;not null;" json:"last_name" validate:"required,min=3"`
	// OTP for user
	OTP int64 `gorm:"type:bigint;not null;" json:"otp"`
	// Bio of the user
	Bio string `gorm:"type:text;size:255;null" json:"bio" validate:"max=255"`
	// Avatar Url of the user
	AvatarUrl string `gorm:"type:text;size:255;null" json:"avatar_url" validate:"url"`
	// Job Title of the user
	JobTitle string `gorm:"size:100;null" json:"job_title" validate:"max=100"`
	// Company of the user
	Employer string `gorm:"size:100;null" json:"employer" validate:"max=100"`
	// Location of the user
	Location string `gorm:"size:100;null" json:"location" validate:"max=100"`
	// GithubURL of the user
	GithubUrl string `gorm:"size:255;" json:"github_url" validate:"url"`
	// Website of the user
	Website string `gorm:"size:255;" json:"website" validate:"url"`

	// Current Learning is for current learning desc
	CurrentLearning string `gorm:"size:200;type:text" json:"current_learning" validate:"max=200"`
	// AvailableFor is what you upto
	AvailableFor string `gorm:"size:200;type:text" json:"available_for" validate:"max=200"`
	// What is you are doing now
	CurrentlyHackingOn string `gorm:"size:200;type:text" json:"currently_hacking_on" validate:"max=200"`
	// Pronouns is for personal attributes to call someone
	Pronouns string `gorm:"size:100;type:text" json:"pronouns" validate:"max=100"`
	// Academic background
	Education string `gorm:"size:100;type:text" json:"education" validate:"max=100"`
	// Brand Color is for color accent for users profile
	BrandColor string `gorm:"size:100;type:text" json:"brand_color" validate:"max=7"`

	// User account activation
	IsActive bool `gorm:"default:false" json:"is_active"`
	// Email verified or not
	IsEmailVerified bool `gorm:"default:false" json:"is_email_verified"`

	// number of posts
	PostsCount int `gorm:"default:0" json:"posts_count"`
	// number of comments
	CommentsCount int `gorm:"default:0" json:"comments_count"`
	// number of likes
	LikesCount int `gorm:"default:0" json:"likes_count"`
	// number of bookmarks
	BookmarksCount int `gorm:"default:0" json:"bookmarks_count"`
	// Show last active time
	LastSeen time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"last_seen"`
	// User roles: user, moderator, admin
	Role string `gorm:"size:50;default:'user'" json:"role"`

	// Skills of the user
	Skills string `gorm:"type:text;size:200" json:"skills" validate:"max=255"`
	// Interests of the user
	Interests []Interest `gorm:"many2many:user_interests;" json:"interests" validate:"max=255"`
	// Badges for users
	Badges []Badge `gorm:"many2many:user_badges;" json:"badges"`
	// Roles for user
	Roles []Role `gorm:"many2many:user_roles" json:"roles"`
	// Followers to follow each other
	Followers []User `gorm:"many2many:user_followers;association_jointable_foreignkey:follower_id" json:"followers"`
	// Who is following
	Following []User `gorm:"many2many:user_followings;association_jointable_foreignkey:following_id" json:"following"`
	// Notification that user got from blog
	Notifications []Notification `gorm:"many2many:user_notifications" json:"notifications"`
	// notifications
	NotificationsPreferences []NotificationPrefrences `gorm:"many2many:user_notifipre" json:"notifipre"`

	// theme preference of the user
	ThemePreference string         `gorm:"default:light" json:"theme_preference"`
	CreatedAt       time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

type Role struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Name   string    `gorm:"size:50;not null;unique" json:"name"` // e.g., "admin", "moderator", "writer"
}

// Interest Model
type Interest struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Name   string    `gorm:"size:100;not null;unique" json:"name"`
}

// Badge Model for User
type Badge struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Name   string    `gorm:"size:100;not null;unique" json:"name"`
}

type UserFollower struct {
	UserID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	FollowerID uuid.UUID `gorm:"type:uuid;primaryKey"`
	FollowedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

type UserFollowing struct {
	UserID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	FollowingID uuid.UUID `gorm:"type:uuid;primaryKey"`
	FollowedAt  time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

// Notification Model
type Notification struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Type      string    `gorm:"size:50;not null" json:"type"` // like, comment, follow, etc.
	Message   string    `gorm:"size:255;not null" json:"message"`
	IsRead    bool      `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
}

type NotificationPrefrences struct {
	// Send notification to user if any event occurs
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	EmailOnLikes    bool      `gorm:"default:false" json:"email_on_likes"`
	EmailOnComments bool      `gorm:"default:false" json:"email_on_comments"`
	EmailOnMentions bool      `gorm:"default:false" json:"email_on_mentions"`
	EmailOnFollower bool      `gorm:"default:false" json:"email_on_followers"`
	EmailOnNewPosts bool      `gorm:"default:false" json:"email_on_new_posts"`
}
