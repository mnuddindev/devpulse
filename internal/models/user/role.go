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

type Role struct {
	ID          uuid.UUID    `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name        string       `gorm:"size:50;not null;unique" json:"name" validate:"required"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions"`
	CreatedAt   time.Time    `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time    `gorm:"autoUpdateTime" json:"updated_at"`
}

// NewRole creates a new role.
func NewRole(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, name string, permissions ...string) (*Role, error) {
	r := &Role{Name: name}
	validate := validator.New()
	if err := validate.Struct(r); err != nil {
		return nil, utils.NewError(utils.ErrBadRequest.Code, "Invalid role data", err.Error())
	}

	if err := db.WithContext(ctx).Create(r).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to create role")
	}

	for _, permName := range permissions {
		var perm Permission
		if err := db.WithContext(ctx).Where("name = ?", permName).FirstOrCreate(&perm).Error; err != nil {
			return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to add permission")
		}
		db.WithContext(ctx).Model(r).Association("Permissions").Append(&perm)
	}

	roleJSON, _ := json.Marshal(r)
	key := "role:" + r.ID.String()
	rclient.Set(ctx, key, roleJSON, 10*time.Minute)
	return r, nil
}

// GetRole retrieves a role by ID.
func GetRole(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, id uuid.UUID) (*Role, error) {
	key := "role:" + id.String()
	if cached, err := rclient.Get(ctx, key).Result(); err == nil {
		var r Role
		if err := json.Unmarshal([]byte(cached), &r); err == nil {
			return &r, nil
		}
	}

	var r Role
	if err := db.WithContext(ctx).Preload("Permissions").Where("id = ?", id).First(&r).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.NewError(utils.ErrNotFound.Code, "Role not found")
		}
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to get role")
	}

	roleJSON, _ := json.Marshal(r)
	rclient.Set(ctx, key, roleJSON, 10*time.Minute)
	return &r, nil
}

// GetRoles retrieves all roles.
func GetRoles(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB) ([]Role, error) {
	key := "roles:all"
	if cached, err := rclient.Get(ctx, key).Result(); err == nil {
		var roles []Role
		if err := json.Unmarshal([]byte(cached), &roles); err == nil {
			return roles, nil
		}
	}

	var roles []Role
	if err := db.WithContext(ctx).Preload("Permissions").Find(&roles).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to get roles")
	}

	rolesJSON, _ := json.Marshal(roles)
	rclient.Set(ctx, key, rolesJSON, 10*time.Minute)
	return roles, nil
}

// UpdateRole updates a role’s name or permissions.
func UpdateRole(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, id uuid.UUID, name string, permissions []string) (*Role, error) {
	r, err := GetRole(ctx, rclient, db, id)
	if err != nil {
		return nil, err
	}

	r.Name = name
	validate := validator.New()
	if err := validate.Struct(r); err != nil {
		return nil, utils.NewError(utils.ErrBadRequest.Code, "Invalid role data", err.Error())
	}

	if err := db.WithContext(ctx).Model(r).Association("Permissions").Clear(); err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to clear permissions")
	}
	for _, permName := range permissions {
		var perm Permission
		if err := db.WithContext(ctx).Where("name = ?", permName).FirstOrCreate(&perm).Error; err != nil {
			return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to add permission")
		}
		db.WithContext(ctx).Model(r).Association("Permissions").Append(&perm)
	}

	if err := db.WithContext(ctx).Save(r).Error; err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update role")
	}

	roleJSON, _ := json.Marshal(r)
	key := "role:" + r.ID.String()
	rclient.Set(ctx, key, roleJSON, 10*time.Minute)
	return r, nil
}

// DeleteRole deletes a role.
func DeleteRole(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, id uuid.UUID) error {
	r, err := GetRole(ctx, rclient, db, id)
	if err != nil {
		return err
	}

	if err := db.WithContext(ctx).Delete(r).Error; err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to delete role")
	}

	key := "role:" + id.String()
	rclient.Del(ctx, key)
	return nil
}

// SeedRoles initializes default roles and permissions.
func SeedRoles(ctx context.Context, db *gorm.DB, rclient *storage.RedisClient) error {
	roles := []struct {
		Name        string
		Permissions []string
	}{
		{"member", []string{
			"read_post", "create_comment", "edit_own_comment", "delete_own_comment",
			"give_reaction", "follow_tag", "unfollow_tag", "follow_user", "unfollow_user",
			"delete_own_profile", "edit_own_profile", "need_moderation",
		}},
		{"author", []string{
			"read_post", "create_post", "edit_own_post", "delete_own_post",
			"create_comment", "edit_own_comment", "delete_own_comment",
			"give_reaction", "follow_tag", "unfollow_tag", "follow_user", "unfollow_user",
			"delete_own_profile", "edit_own_profile",
		}},
		{"trusted_member", []string{
			"read_post", "create_post", "edit_own_post", "delete_own_post",
			"create_comment", "edit_own_comment", "delete_own_comment",
			"give_reaction", "follow_tag", "unfollow_tag", "follow_user", "unfollow_user",
			"delete_own_profile", "edit_own_profile", "give_suggestion",
		}},
		{"tag_moderator", []string{
			"read_post", "create_post", "edit_own_post", "delete_own_post",
			"create_comment", "edit_own_comment", "delete_own_comment",
			"give_reaction", "follow_tag", "unfollow_tag", "follow_user", "unfollow_user",
			"delete_own_profile", "edit_own_profile", "give_suggestion",
			"moderate_tag", "feature_posts", "ban_user",
		}},
		{"moderator", []string{
			"read_post", "create_post", "edit_own_post", "delete_own_post", "edit_any_post",
			"delete_any_post", "moderate_post", "create_comment", "edit_own_comment",
			"delete_own_comment", "edit_any_comment", "moderate_comment", "create_user",
			"edit_user", "delete_user", "moderate_user", "ban_user", "follow_user",
			"unfollow_user", "create_roles", "edit_roles", "delete_roles", "create_reaction",
			"edit_reaction", "delete_reaction", "give_reaction", "create_tag", "edit_tag",
			"delete_tag", "moderate_tag", "follow_tag", "unfollow_tag", "give_suggestion",
			"feature_posts",
		}},
		{"admin", []string{
			"read_post", "create_post", "edit_own_post", "delete_own_post", "edit_any_post",
			"delete_any_post", "moderate_post", "create_comment", "edit_own_comment",
			"delete_own_comment", "edit_any_comment", "moderate_comment", "create_user",
			"edit_user", "delete_user", "moderate_user", "ban_user", "follow_user",
			"unfollow_user", "create_roles", "edit_roles", "delete_roles", "create_reaction",
			"edit_reaction", "delete_reaction", "give_reaction", "create_tag", "edit_tag",
			"delete_tag", "moderate_tag", "follow_tag", "unfollow_tag", "give_suggestion",
			"feature_posts", "assign_roles", "site_settings", // Extra admin powers
		}},
	}

	for _, r := range roles {
		var role Role
		if err := db.WithContext(ctx).Where("name = ?", r.Name).FirstOrCreate(&role).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to seed role: "+r.Name)
		}

		for _, permName := range r.Permissions {
			var perm Permission
			if err := db.WithContext(ctx).Where("name = ?", permName).FirstOrCreate(&perm).Error; err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to seed permission: "+permName)
			}
			db.WithContext(ctx).Model(&role).Association("Permissions").Append(&perm)
		}

		// Cache in Redis
		var perms []string
		db.WithContext(ctx).Model(&role).Association("Permissions").Find(&perms)
		permsJSON, _ := json.Marshal(perms)
		rclient.Set(ctx, "perms:role:"+r.Name, permsJSON, 10*time.Minute)
	}

	return nil
}
