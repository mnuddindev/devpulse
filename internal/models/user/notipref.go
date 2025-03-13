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
)

type NotificationPreferences struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;unique" json:"user_id"`
	EmailOnLikes     bool      `gorm:"default:false" json:"email_on_likes"`
	EmailOnComments  bool      `gorm:"default:false" json:"email_on_comments"`
	EmailOnMentions  bool      `gorm:"default:false" json:"email_on_mentions"`
	EmailOnFollowers bool      `gorm:"default:false" json:"email_on_followers"`
	EmailOnBadge     bool      `gorm:"default:false" json:"email_on_badge"`
	EmailOnUnread    bool      `gorm:"default:false" json:"email_on_unread"`
	EmailOnNewPosts  bool      `gorm:"default:false" json:"email_on_new_posts"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// NewNotificationPreferences creates preferences for a user.
func NewNotificationPreferences(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, userID uuid.UUID) (*NotificationPreferences, error) {
	np := &NotificationPreferences{UserID: userID}
	validate := validator.New()
	if err := validate.Struct(np); err != nil {
		return nil, utils.NewError(utils.ErrBadRequest.Code, "Invalid notification preferences data", err.Error())
	}

	if err := gormDB.WithContext(ctx).Create(np).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create notification preferences")
	}

	npJSON, _ := json.Marshal(np)
	key := "notif_prefs:" + np.ID.String()
	redisClient.Set(ctx, key, npJSON, 10*time.Minute)
	return np, nil
}

// GetNotificationPreferences retrieves preferences by ID.
func GetNotificationPreferences(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID) (*NotificationPreferences, error) {
	key := "notif_prefs:" + id.String()
	if cached, err := redisClient.Get(ctx, key).Result(); err == nil {
		var np NotificationPreferences
		if err := json.Unmarshal([]byte(cached), &np); err == nil {
			return &np, nil
		}
	}

	var np NotificationPreferences
	if err := gormDB.WithContext(ctx).Where("id = ?", id).First(&np).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Notification preferences not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to get notification preferences")
	}

	npJSON, _ := json.Marshal(np)
	redisClient.Set(ctx, key, npJSON, 10*time.Minute)
	return &np, nil
}

// GetNotificationPreferencesByUser retrieves preferences by user ID.
func GetNotificationPreferencesByUser(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, userID uuid.UUID) (*NotificationPreferences, error) {
	key := "notif_prefs:user:" + userID.String()
	if cached, err := redisClient.Get(ctx, key).Result(); err == nil {
		var np NotificationPreferences
		if err := json.Unmarshal([]byte(cached), &np); err == nil {
			return &np, nil
		}
	}

	var np NotificationPreferences
	if err := gormDB.WithContext(ctx).Where("user_id = ?", userID).First(&np).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Notification preferences not found for user")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to get notification preferences")
	}

	npJSON, _ := json.Marshal(np)
	redisClient.Set(ctx, key, npJSON, 10*time.Minute)
	return &np, nil
}

// UpdateNotificationPreferences updates preferences.
func UpdateNotificationPreferences(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID, likes, comments, mentions, followers, badge, unread, newPosts bool) (*NotificationPreferences, error) {
	np, err := GetNotificationPreferences(ctx, redisClient, gormDB, id)
	if err != nil {
		return nil, err
	}

	np.EmailOnLikes = likes
	np.EmailOnComments = comments
	np.EmailOnMentions = mentions
	np.EmailOnFollowers = followers
	np.EmailOnBadge = badge
	np.EmailOnUnread = unread
	np.EmailOnNewPosts = newPosts

	if err := gormDB.WithContext(ctx).Save(np).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update notification preferences")
	}

	npJSON, _ := json.Marshal(np)
	key := "notif_prefs:" + np.ID.String()
	redisClient.Set(ctx, key, npJSON, 10*time.Minute)
	return np, nil
}

// DeleteNotificationPreferences deletes preferences.
func DeleteNotificationPreferences(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID) error {
	np, err := GetNotificationPreferences(ctx, redisClient, gormDB, id)
	if err != nil {
		return err
	}

	if err := gormDB.WithContext(ctx).Delete(np).Error; err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to delete notification preferences")
	}

	key := "notif_prefs:" + id.String()
	redisClient.Del(ctx, key)
	return nil
}
