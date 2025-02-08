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
	AvatarUrl string `gorm:"type:text;size:255;null" json:"avatar_url"`
	// Job Title of the user
	JobTitle string `gorm:"size:100;null" json:"job_title" validate:"max=100"`
	// Company of the user
	Employer string `gorm:"size:100;null" json:"employer" validate:"max=100"`
	// Location of the user
	Location string `gorm:"size:100;null" json:"location" validate:"max=100"`
	// GithubURL of the user
	GithubUrl string `gorm:"size:255;" json:"github_url"`
	// Website of the user
	Website string `gorm:"size:255;" json:"website"`

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
	Skills []Skill `gorm:"many2many:user_skills;" json:"skills" validate:"max=255"`
	// Interests of the user
	Interests []Interest `gorm:"many2many:user_interests;" json:"interests" validate:"max=255"`
	// Badges for users
	Badges []Badge `gorm:"many2many:user_badges;" json:"badges"`
	// Roles for user
	Roles []Role `gorm:"many2many:user_roles" json:"roles"`
	// Followers to follow each other
	Followers []string `gorm:"type:text[]" json:"followers"`
	// Who is following
	Following []string `gorm:"type:text[]" json:"following"`
	// Notification that user got from blog
	Notifications []Notification `gorm:"foreignKey:UserID" json:"notifications"`
	// notifications
	NotificationsPreferences []NotificationPrefrences `gorm:"foreignkey:UserID" json:"notifipre"`

	// theme preference of the user
	ThemePreference string         `gorm:"default:light" json:"theme_preference"`
	CreatedAt       time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

type Role struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name string    `gorm:"size:50;not null;unique" json:"name"` // e.g., "admin", "moderator", "writer"
}

// Skill Model
type Skill struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name string    `gorm:"size:100;not null;unique" json:"name"`
}

// Interest Model
type Interest struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name string    `gorm:"size:100;not null;unique" json:"name"`
}

// SocialLinks Model for User
type SocialLinks struct {
	Twitter   string `gorm:"size:255;null" json:"twitter"`
	LinkedIn  string `gorm:"size:255;null" json:"linkedin"`
	Instagram string `gorm:"size:255;null" json:"instagram"`
}

// Badge Model for User
type Badge struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name string    `gorm:"size:100;not null;unique" json:"name"`
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
	UserID          uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	EmailOnLikes    bool      `gorm:"default:false" json:"email_on_likes"`
	EmailOnComments bool      `gorm:"default:false" json:"email_on_comments"`
	EmailOnMentions bool      `gorm:"default:false" json:"email_on_mentions"`
	EmailOnFollower bool      `gorm:"default:false" json:"email_on_followers"`
	EmailOnNewPosts bool      `gorm:"default:false" json:"email_on_new_posts"`
}
