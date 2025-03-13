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

type Permission struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"size:50;not null;unique" json:"name"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// NewPermission creates a new permission.
func NewPermission(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, name string) (*Permission, error) {
	p := &Permission{Name: name}
	validate := validator.New()
	if err := validate.Struct(p); err != nil {
		return nil, utils.NewError(utils.ErrBadRequest.Code, "Invalid permission data", err.Error())
	}

	if err := gormDB.WithContext(ctx).Create(p).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create permission")
	}

	permJSON, _ := json.Marshal(p)
	key := "permission:" + p.ID.String()
	redisClient.Set(ctx, key, permJSON, 10*time.Minute)
	return p, nil
}

// GetPermission retrieves a permission by ID.
func GetPermission(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID) (*Permission, error) {
	key := "permission:" + id.String()
	if cached, err := redisClient.Get(ctx, key).Result(); err == nil {
		var p Permission
		if err := json.Unmarshal([]byte(cached), &p); err == nil {
			return &p, nil
		}
	}

	var p Permission
	if err := gormDB.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Permission not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to get permission")
	}

	permJSON, _ := json.Marshal(p)
	redisClient.Set(ctx, key, permJSON, 10*time.Minute)
	return &p, nil
}

// GetPermissions retrieves all permissions.
func GetPermissions(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB) ([]Permission, error) {
	key := "permissions:all"
	if cached, err := redisClient.Get(ctx, key).Result(); err == nil {
		var perms []Permission
		if err := json.Unmarshal([]byte(cached), &perms); err == nil {
			return perms, nil
		}
	}

	var perms []Permission
	if err := gormDB.WithContext(ctx).Find(&perms).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to get permissions")
	}

	permsJSON, _ := json.Marshal(perms)
	redisClient.Set(ctx, key, permsJSON, 10*time.Minute)
	return perms, nil
}

// UpdatePermission updates a permission’s name.
func UpdatePermission(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID, name string) (*Permission, error) {
	p, err := GetPermission(ctx, redisClient, gormDB, id)
	if err != nil {
		return nil, err
	}

	p.Name = name
	validate := validator.New()
	if err := validate.Struct(p); err != nil {
		return nil, utils.NewError(utils.ErrBadRequest.Code, "Invalid permission data", err.Error())
	}

	if err := gormDB.WithContext(ctx).Save(p).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update permission")
	}

	permJSON, _ := json.Marshal(p)
	key := "permission:" + p.ID.String()
	redisClient.Set(ctx, key, permJSON, 10*time.Minute)
	return p, nil
}

// DeletePermission deletes a permission.
func DeletePermission(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, id uuid.UUID) error {
	p, err := GetPermission(ctx, redisClient, gormDB, id)
	if err != nil {
		return err
	}

	if err := gormDB.WithContext(ctx).Delete(p).Error; err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to delete permission")
	}

	key := "permission:" + id.String()
	redisClient.Del(ctx, key)
	return nil
}
