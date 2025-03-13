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

	Username        string    `gorm:"size:255;not null;unique" json:"username" validate:"required,min=3,max=255,alphanum"`
	Email           string    `gorm:"size:100;not null;unique" json:"email" validate:"required,email"`
	Password        string    `gorm:"size:255;not null" json:"-" validate:"required,min=6"`
	OTP             int64     `gorm:"type:bigint;not null" json:"otp"`
	IsActive        bool      `gorm:"default:false" json:"is_active"`
	IsEmailVerified bool      `gorm:"default:false" json:"is_email_verified"`
	RoleID          uuid.UUID `gorm:"type:uuid;not null" json:"role_id"`
	Role            Role      `gorm:"foreignKey:RoleID" json:"role"`

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

	var memberRole Role
	if err := db.WithContext(ctx).Where("name = ?", "member").First(&memberRole).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Default 'member' not found!!")
	}

	otp, err := utils.GenerateOTP()
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "OTP generation failed.")
	}

	u := &User{
		Username: username,
		Email:    email,
		Password: password,
		OTP:      otp,
		RoleID:   memberRole.ID,
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

// GetUser retrieves a user by ID, with optional preloading of relationships.
func GetUser(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID, preload ...string) (*User, error) {
	key := "user:" + id.String()
	if cached, err := redisClient.Get(ctx, key).Result(); err == nil {
		var u User
		if err := json.Unmarshal([]byte(cached), &u); err == nil {
			return &u, nil
		}
	}

	var u User
	query := gormDB.WithContext(ctx).Where("id = ?", id)
	for _, p := range preload {
		query = query.Preload(p)
	}
	if err := query.First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "User not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to get user")
	}

	userJSON, _ := json.Marshal(u)
	redisClient.Set(ctx, key, userJSON, 10*time.Minute)
	return &u, nil
}

// GetUsers retrieves multiple users with pagination and optional filters.
func GetUsers(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, page, limit int, filters ...string) ([]User, error) {
	key := "users:page:" + string(page) + ":limit:" + string(limit)
	if cached, err := redisClient.Get(ctx, key).Result(); err == nil {
		var users []User
		if err := json.Unmarshal([]byte(cached), &users); err == nil {
			return users, nil
		}
	}

	var users []User
	query := gormDB.WithContext(ctx).Offset((page - 1) * limit).Limit(limit)
	for _, f := range filters {
		query = query.Where(f)
	}
	if err := query.Find(&users).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to get users")
	}

	usersJSON, _ := json.Marshal(users)
	redisClient.Set(ctx, key, usersJSON, 10*time.Minute)
	return users, nil
}

// UpdateUser updates a user’s fields and refreshes cache.
func UpdateUser(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID, opts ...UserOption) (*User, error) {
	u, err := GetUser(ctx, redisClient, gormDB, id)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(u)
	}

	validate := validator.New()
	if err := validate.Struct(u); err != nil {
		return nil, utils.NewError(utils.ErrBadRequest.Code, "Invalid user data", err.Error())
	}

	if err := gormDB.WithContext(ctx).Save(u).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update user")
	}

	userJSON, _ := json.Marshal(u)
	key := "user:" + u.ID.String()
	redisClient.Set(ctx, key, userJSON, 10*time.Minute)
	return u, nil
}

// DeleteUser soft-deletes a user and clears cache.
func DeleteUser(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID) error {
	u, err := GetUser(ctx, redisClient, gormDB, id)
	if err != nil {
		return err
	}

	if err := gormDB.WithContext(ctx).Delete(u).Error; err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to delete user")
	}

	key := "user:" + id.String()
	redisClient.Del(ctx, key)
	return nil
}

// VerifyEmail marks a user’s email as verified if OTP matches.
func (u *User) VerifyEmail(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, otp int64) error {
	if u.OTP != otp {
		return utils.NewError(utils.ErrBadRequest.Code, "Invalid OTP")
	}

	u.IsEmailVerified = true
	u.OTP = 0 // Reset OTP after verification
	if err := gormDB.WithContext(ctx).Save(u).Error; err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to verify email")
	}

	userJSON, _ := json.Marshal(u)
	key := "user:" + u.ID.String()
	redisClient.Set(ctx, key, userJSON, 10*time.Minute)
	return nil
}

// FollowUser adds a follow relationship.
func (u *User) FollowUser(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, followID uuid.UUID) error {
	followee, err := GetUser(ctx, redisClient, gormDB, followID)
	if err != nil {
		return err
	}
	if u.ID == followee.ID {
		return utils.NewError(utils.ErrBadRequest.Code, "Cannot follow yourself")
	}
	if err := gormDB.WithContext(ctx).Model(u).Association("Following").Append(followee); err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to follow user")
	}
	userJSON, _ := json.Marshal(u)
	key := "user:" + u.ID.String()
	redisClient.Set(ctx, key, userJSON, 10*time.Minute)
	return nil
}

// UnfollowUser removes a user from the following list.
func (u *User) UnfollowUser(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, followID uuid.UUID) error {
	followee, err := GetUser(ctx, redisClient, gormDB, followID)
	if err != nil {
		return err
	}

	if err := gormDB.WithContext(ctx).Model(u).Association("Following").Delete(followee); err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to unfollow user")
	}

	userJSON, _ := json.Marshal(u)
	key := "user:" + u.ID.String()
	redisClient.Set(ctx, key, userJSON, 10*time.Minute)
	return nil
}

// UpdateLastSeen refreshes the user’s last seen timestamp.
func (u *User) UpdateLastSeen(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB) error {
	u.Stats.LastSeen = time.Now()
	if err := gormDB.WithContext(ctx).Save(u).Error; err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update last seen")
	}

	userJSON, _ := json.Marshal(u)
	key := "user:" + u.ID.String()
	redisClient.Set(ctx, key, userJSON, 10*time.Minute)
	return nil
}

// HasPermission checks if the user has a permission.
func (u *User) HasPermission(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, permission string) bool {
	cacheKey := "perms:role:" + u.RoleID.String()
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
	if err := db.WithContext(ctx).Preload("Permissions").Where("id = ?", u.RoleID).First(&role).Error; err != nil {
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
