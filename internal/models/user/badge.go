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

type Badge struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"size:100;not null;unique" json:"name"`
	Image     string    `gorm:"size:255;not null" json:"image" validate:"required,url"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// NewBadge creates a new badge.
func NewBadge(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, name, image string) (*Badge, error) {
	b := &Badge{Name: name, Image: image}
	validate := validator.New()
	if err := validate.Struct(b); err != nil {
		return nil, utils.NewError(utils.ErrBadRequest.Code, "Invalid badge data", err.Error())
	}

	if err := gormDB.WithContext(ctx).Create(b).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create badge")
	}

	badgeJSON, _ := json.Marshal(b)
	key := "badge:" + b.ID.String()
	redisClient.Set(ctx, key, badgeJSON, 10*time.Minute)
	return b, nil
}

// GetBadge retrieves a badge by ID.
func GetBadge(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID) (*Badge, error) {
	key := "badge:" + id.String()
	if cached, err := redisClient.Get(ctx, key).Result(); err == nil {
		var b Badge
		if err := json.Unmarshal([]byte(cached), &b); err == nil {
			return &b, nil
		}
	}

	var b Badge
	if err := gormDB.WithContext(ctx).Where("id = ?", id).First(&b).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Badge not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to get badge")
	}

	badgeJSON, _ := json.Marshal(b)
	redisClient.Set(ctx, key, badgeJSON, 10*time.Minute)
	return &b, nil
}

// GetBadges retrieves all badges.
func GetBadges(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB) ([]Badge, error) {
	key := "badges:all"
	if cached, err := redisClient.Get(ctx, key).Result(); err == nil {
		var badges []Badge
		if err := json.Unmarshal([]byte(cached), &badges); err == nil {
			return badges, nil
		}
	}

	var badges []Badge
	if err := gormDB.WithContext(ctx).Find(&badges).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to get badges")
	}

	badgesJSON, _ := json.Marshal(badges)
	redisClient.Set(ctx, key, badgesJSON, 10*time.Minute)
	return badges, nil
}

// UpdateBadge updates a badge.
func UpdateBadge(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID, name, image string) (*Badge, error) {
	b, err := GetBadge(ctx, redisClient, gormDB, id)
	if err != nil {
		return nil, err
	}

	b.Name = name
	b.Image = image
	validate := validator.New()
	if err := validate.Struct(b); err != nil {
		return nil, utils.NewError(utils.ErrBadRequest.Code, "Invalid badge data", err.Error())
	}

	if err := gormDB.WithContext(ctx).Save(b).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update badge")
	}

	badgeJSON, _ := json.Marshal(b)
	key := "badge:" + b.ID.String()
	redisClient.Set(ctx, key, badgeJSON, 10*time.Minute)
	return b, nil
}

// DeleteBadge deletes a badge.
func DeleteBadge(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID) error {
	b, err := GetBadge(ctx, redisClient, gormDB, id)
	if err != nil {
		return err
	}

	if err := gormDB.WithContext(ctx).Delete(b).Error; err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to delete badge")
	}

	key := "badge:" + id.String()
	redisClient.Del(ctx, key)
	return nil
}

// AddBadgeToUser assigns a badge to a user (utility).
func AddBadgeToUser(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, userID, badgeID uuid.UUID) error {
	u, err := GetUserBy(ctx, redisClient, gormDB, "id = ?", []interface{}{userID}, "")
	if err != nil {
		return err
	}
	b, err := GetBadge(ctx, redisClient, gormDB, badgeID)
	if err != nil {
		return err
	}

	if err := gormDB.WithContext(ctx).Model(u).Association("Badges").Append(b); err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to add badge to user")
	}

	userJSON, _ := json.Marshal(u)
	key := "user:" + u.ID.String()
	redisClient.Set(ctx, key, userJSON, 10*time.Minute)
	return nil
}
