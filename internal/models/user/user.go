package models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id" validate:"requried"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt time.Time `gorm:"index" json:"-"`

	Username        string `gorm:"size:255;not null;unique" json:"username" validate:"required,min=3,max=255,alphanum"`
	Email           string `gorm:"size:100;not null;unique" json:"email" validate:"required,email"`
	Password        string `gorm:"size:255;not null" json:"-" validate:"required,min=6"`
	OTP             int64  `gorm:"type:bigint;not null" json:"otp"`
	IsActive        bool   `gorm:"default:false" json:"is_active"`
	IsEmailVerified bool   `gorm:"default:false" json:"is_email_verified"`
	Role            Role   `gorm:"foreignKey:Name;references:Name" json:"role"`

	Profile struct {
		Name               string `gorm:"size:100" json:"name" validate:"omitempty,max=100"`
		Bio                string `gorm:"type:text;size:255" json:"bio" validate:"omitempty,max=255"`
		AvatarURL          string `gorm:"type:text;size:255" json:"avatar_url" validate:"omitempty,url"`
		JobTitle           string `gorm:"size:100" json:"job_title" validate:"omitempty,max=100"`
		Employer           string `gorm:"size:100" json:"employer" validate:"omitempty,max=100"`
		Location           string `gorm:"size:100" json:"location" validate:"omitempty,max=100"`
		SocialLinks        string `gorm:"type:jsonb;default:'{}'" json:"social_links" validate:"omitempty"`
		CurrentLearning    string `gorm:"type:text;size:200" json:"current_learning" validate:"omitempty,max=200"`
		AvailableFor       string `gorm:"type:text;size:200" json:"available_for" validate:"omitempty,max=200"`
		CurrentlyHackingOn string `gorm:"type:text;size:200" json:"currently_hacking_on" validate:"omitempty,max=200"`
		Pronouns           string `gorm:"type:text;size:100" json:"pronouns" validate:"omitempty,max=100"`
		Education          string `gorm:"type:text;size:100" json:"education" validate:"omitempty,max=100"`
		Skills             string `gorm:"type:jsonb;default:'[]'" json:"skills" validate:"omitempty"`
		Interests          string `gorm:"type:jsonb;default:'[]'" json:"interests" validate:"omitempty"`
	} `gorm:"embedded"`

	Settings struct {
		BrandColor      string `gorm:"type:text;size:7" json:"brand_color" validate:"omitempty,max=7"`
		ThemePreference string `gorm:"size:20;default:'light'" json:"theme_preference" validate:"oneof=light dark"`
		BaseFont        string `gorm:"size:50;default:'sans-serif'" json:"base_font" validate:"oneof=sans-serif sans jetbrainsmono hind-siliguri comic-sans"`
		SiteNavbar      string `gorm:"size:20;default:'fixed'" json:"site_navbar" validate:"oneof=fixed static"`
		ContentEditor   string `gorm:"size:20;default:'rich'" json:"content_editor" validate:"oneof=rich basic"`
		ContentMode     int    `gorm:"default:1" json:"content_mode" validate:"oneof=1 2 3 4 5"`
	} `gorm:"embedded"`

	Stats struct {
		PostsCount     int       `gorm:"default:0" json:"posts_count"`
		CommentsCount  int       `gorm:"default:0" json:"comments_count"`
		LikesCount     int       `gorm:"default:0" json:"likes_count"`
		BookmarksCount int       `gorm:"default:0" json:"bookmarks_count"`
		LastSeen       time.Time `gorm:"default:current_timestamp" json:"last_seen"`
	} `gorm:"embedded"`

	Badges                  []Badge                 `gorm:"many2many:user_badges;" json:"badges"`
	Followers               []User                  `gorm:"many2many:user_followers;joinForeignKey:following_id;joinReferences:follower_id" json:"followers"`
	Following               []User                  `gorm:"many2many:user_followers;joinForeignKey:follower_id;joinReferences:following_id" json:"following"`
	Notifications           []Notification          `gorm:"foreignKey:UserID" json:"notifications"`
	NotificationPreferences NotificationPreferences `gorm:"foreignKey:UserID" json:"notification_preferences"`
}

// UserOption configures a User.
type UserOption func(*User)

// NewUser creates a new User instance with validation.
func NewUser(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, username, email, password string, opts ...UserOption) (*User, error) {
	if err := ctx.Err(); err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "user credential canceled")
	}

	u := &User{
		Username: username,
		Email:    email,
		Password: password,
		Role:     Role{Name: "member"},
	}

	for _, opt := range opts {
		opt(u)
	}

	validate := validator.New()
	if err := validate.Struct(u); err != nil {
		return nil, utils.NewError(utils.ErrBadRequest.Code, "Invalid user data", err.Error())
	}

	if err := db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create user in database")
	}

	userJSON, _ := json.Marshal(u)
	key := "user:" + u.ID.String()
	if err := rclient.Set(ctx, key, userJSON, 10*time.Minute).Err(); err != nil {
		logger.Default.Warn(ctx, "Failed to cache user in Redis: %v", err)
	}

	return u, nil
}

// HasPermission checks if the user has a permission.
func (u *User) HasPermission(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, permission string) bool {
	cacheKey := "perms:role:" + u.Role.Name
	if cachedPerms, err := rclient.Get(ctx, cacheKey).Result(); err == nil {
		var perms []string
		if json.Unmarshal([]byte(cachedPerms), &perms) == nil {
			for _, p := range perms {
				if p == permission {
					return true
				}
			}
			return false
		}
	}

	var role Role
	if err := db.WithContext(ctx).Preload("Permissions").Where("name = ?", u.Role.Name).First(&role).Error; err != nil {
		return false
	}

	for _, p := range role.Permissions {
		if p.Name == permission {
			perms := make([]string, len(role.Permissions))
			for i, perm := range role.Permissions {
				perms[i] = perm.Name
			}
			if permsJSON, err := json.Marshal(perms); err == nil {
				rclient.Set(ctx, cacheKey, permsJSON, 10*time.Minute)
			}
			return true
		}
	}
	return false
}
