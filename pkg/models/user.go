package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Username           string         `gorm:"size:255;not null;unique" json:"username" validate:"required,min=3"`
	Email              string         `gorm:"size:100;not null;unique" json:"email" validate:"required,email"`
	Password           string         `gorm:"not null;" json:"password" validate:"required,min=6"`
	FirstName          string         `gorm:"size:100;not null;" json:"first_name" validate:"required,min=3"`
	LastName           string         `gorm:"size:100;not null;" json:"last_name" validate:"required,min=3"`
	OTP                int64          `gorm:"type:bigint;not null;" json:"otp"`
	Bio                string         `gorm:"type:text;size:255;null" json:"bio" validate:"max=255"`
	AvatarUrl          string         `gorm:"type:text;size:255;null" json:"avatar_url" validate:"omitempty,url"`
	JobTitle           string         `gorm:"size:100;null" json:"job_title" validate:"max=100"`
	Employer           string         `gorm:"size:100;null" json:"employer" validate:"max=100"`
	Location           string         `gorm:"size:100;null" json:"location" validate:"max=100"`
	GithubUrl          string         `gorm:"size:255;" json:"github_url" validate:"omitempty,url"`
	Website            string         `gorm:"size:255;" json:"website" validate:"omitempty,url"`
	CurrentLearning    string         `gorm:"size:200;type:text" json:"current_learning" validate:"max=200"`
	AvailableFor       string         `gorm:"size:200;type:text" json:"available_for" validate:"max=200"`
	CurrentlyHackingOn string         `gorm:"size:200;type:text" json:"currently_hacking_on" validate:"max=200"`
	Pronouns           string         `gorm:"size:100;type:text" json:"pronouns" validate:"max=100"`
	Education          string         `gorm:"size:100;type:text" json:"education" validate:"max=100"`
	BrandColor         string         `gorm:"size:100;type:text" json:"brand_color" validate:"max=7"`
	IsActive           bool           `gorm:"default:false" json:"is_active"`
	IsEmailVerified    bool           `gorm:"default:false" json:"is_email_verified"`
	PostsCount         int            `gorm:"default:0" json:"posts_count"`
	CommentsCount      int            `gorm:"default:0" json:"comments_count"`
	LikesCount         int            `gorm:"default:0" json:"likes_count"`
	BookmarksCount     int            `gorm:"default:0" json:"bookmarks_count"`
	LastSeen           time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"last_seen"`
	Role               string         `gorm:"size:50;default:'user'" json:"role"`
	ThemePreference    string         `gorm:"default:light" json:"theme_preference"`
	CreatedAt          time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`

	Skills                   string                   `gorm:"type:text" json:"skills"`
	Interests                string                   `gorm:"type:text" json:"interests"`
	Badges                   []Badge                  `gorm:"many2many:user_badges;" json:"badges"`
	Roles                    []Role                   `gorm:"many2many:user_roles;" json:"roles"`
	Followers                []User                   `gorm:"many2many:user_followers;association_jointable_foreignkey:follower_id" json:"followers"`
	Following                []User                   `gorm:"many2many:user_following;association_jointable_foreignkey:following_id" json:"following"`
	Notifications            []Notification           `gorm:"foreignKey:UserID" json:"notifications"`
	NotificationsPreferences []NotificationPrefrences `gorm:"foreignKey:UserID" json:"notifipre"`
}

type Skill struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type Role struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"size:50;not null;unique" json:"name"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type Interest struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"size:100;not null;unique" json:"name"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type Badge struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"size:100;not null;unique" json:"name"`
	Image     string    `gorm:"size:100;not null" json:"image"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type BaseReadingFont struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;unique" json:"user_id" validate:"required"`
	FontSize      int       `gorm:"type:int;not null;default:16" json:"font_size" validate:"gte=10,lte=72"`
	FontStyle     string    `gorm:"type:varchar(50);not null;default:'sans-serif'" json:"font_style" validate:"oneof=sans-serif serif monospace cursive fantasy"`
	LineHeight    float64   `gorm:"type:numeric;not null;default:1.5" json:"line_height" validate:"gte=1.0,lte=3.0"`
	LetterSpacing float64   `gorm:"type:numeric;not null;default:0.0" json:"letter_spacing" validate:"gte=0.0,lte=5.0"`
	CreatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type Notification struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Type      string    `gorm:"size:50;not null" json:"type"`
	Message   string    `gorm:"size:255;not null" json:"message"`
	IsRead    bool      `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
}

type NotificationPrefrences struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	EmailOnLikes    bool      `gorm:"default:false" json:"email_on_likes"`
	EmailOnComments bool      `gorm:"default:false" json:"email_on_comments"`
	EmailOnMentions bool      `gorm:"default:false" json:"email_on_mentions"`
	EmailOnFollower bool      `gorm:"default:false" json:"email_on_followers"`
	EmailOnBadge    bool      `gorm:"default:false" json:"email_on_badge"`
	EmailOnUnread   bool      `gorm:"default:false" json:"email_on_unread"`
	EmailOnNewPosts bool      `gorm:"default:false" json:"email_on_new_posts"`
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type UpdateUser struct {
	Username           *string   `json:"username" validate:"omitempty,min=3"`
	Email              *string   `json:"email" validate:"omitempty,email"`
	FirstName          *string   `json:"first_name" validate:"omitempty,min=3"`
	LastName           *string   `json:"last_name" validate:"omitempty,min=3"`
	Bio                *string   `json:"bio" validate:"omitempty,max=255"`
	AvatarUrl          *string   `json:"avatar_url" validate:"omitempty,url"`
	JobTitle           *string   `json:"job_title" validate:"omitempty,max=100"`
	Employer           *string   `json:"employer" validate:"omitempty,max=100"`
	Location           *string   `json:"location" validate:"omitempty,max=100"`
	GithubUrl          *string   `json:"github_url" validate:"omitempty,url"`
	Website            *string   `json:"website" validate:"omitempty,url"`
	CurrentLearning    *string   `json:"current_learning" validate:"omitempty,max=200"`
	AvailableFor       *string   `json:"available_for" validate:"omitempty,max=200"`
	CurrentlyHackingOn *string   `json:"currently_hacking_on" validate:"omitempty,max=200"`
	Pronouns           *string   `json:"pronouns" validate:"omitempty,max=100"`
	Education          *string   `json:"education" validate:"omitempty,max=100"`
	BrandColor         *string   `json:"brand_color" validate:"omitempty,max=7"`
	ThemePreference    *string   `json:"theme_preference" validate:"omitempty,oneof=light dark"`
	Skills             *string   `json:"skills" validate:"omitempty"`
	Interests          *string   `json:"interests" validate:"omitempty"`
	UpdatedAt          time.Time `json:"updated_at"`
}
