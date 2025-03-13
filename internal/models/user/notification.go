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

type Notification struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Type      string    `gorm:"size:50;not null" json:"type"`
	Message   string    `gorm:"size:255;not null" json:"message"`
	IsRead    bool      `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// NewNotification creates a new notification.
func NewNotification(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, userID uuid.UUID, notifType, message string) (*Notification, error) {
	n := &Notification{UserID: userID, Type: notifType, Message: message}
	validate := validator.New()
	if err := validate.Struct(n); err != nil {
		return nil, utils.NewError(utils.ErrBadRequest.Code, "Invalid notification data", err.Error())
	}

	if err := gormDB.WithContext(ctx).Create(n).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create notification")
	}

	notifJSON, _ := json.Marshal(n)
	key := "notification:" + n.ID.String()
	redisClient.Set(ctx, key, notifJSON, 10*time.Minute)
	return n, nil
}

// GetNotification retrieves a notification by ID.
func GetNotification(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID) (*Notification, error) {
	key := "notification:" + id.String()
	if cached, err := redisClient.Get(ctx, key).Result(); err == nil {
		var n Notification
		if err := json.Unmarshal([]byte(cached), &n); err == nil {
			return &n, nil
		}
	}

	var n Notification
	if err := gormDB.WithContext(ctx).Where("id = ?", id).First(&n).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Notification not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to get notification")
	}

	notifJSON, _ := json.Marshal(n)
	redisClient.Set(ctx, key, notifJSON, 10*time.Minute)
	return &n, nil
}

// GetNotifications retrieves a user’s notifications.
func GetNotifications(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, userID uuid.UUID, page, limit int) ([]Notification, error) {
	key := "notifications:user:" + userID.String() + ":page:" + string(rune(page)) + ":limit:" + string(rune(limit))
	if cached, err := redisClient.Get(ctx, key).Result(); err == nil {
		var notifs []Notification
		if err := json.Unmarshal([]byte(cached), &notifs); err == nil {
			return notifs, nil
		}
	}

	var notifs []Notification
	if err := gormDB.WithContext(ctx).Where("user_id = ?", userID).Offset((page - 1) * limit).Limit(limit).Find(&notifs).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to get notifications")
	}

	notifsJSON, _ := json.Marshal(notifs)
	redisClient.Set(ctx, key, notifsJSON, 10*time.Minute)
	return notifs, nil
}

// UpdateNotification updates a notification (e.g., mark as read).
func UpdateNotification(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID, isRead bool) (*Notification, error) {
	n, err := GetNotification(ctx, redisClient, gormDB, id)
	if err != nil {
		return nil, err
	}

	n.IsRead = isRead
	if err := gormDB.WithContext(ctx).Save(n).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update notification")
	}

	notifJSON, _ := json.Marshal(n)
	key := "notification:" + n.ID.String()
	redisClient.Set(ctx, key, notifJSON, 10*time.Minute)
	return n, nil
}

// DeleteNotification deletes a notification.
func DeleteNotification(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID) error {
	n, err := GetNotification(ctx, redisClient, gormDB, id)
	if err != nil {
		return err
	}

	if err := gormDB.WithContext(ctx).Delete(n).Error; err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to delete notification")
	}

	key := "notification:" + id.String()
	redisClient.Del(ctx, key)
	return nil
}
