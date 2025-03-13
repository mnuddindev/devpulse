package models

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
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
func SeedRoles(ctx context.Context, db *gorm.DB, redisClient *storage.RedisClient, logger *logger.Logger) error {
	allPermissions := []string{
		// Post-related
		"create_post", "delete_any_post", "delete_own_post", "edit_any_post", "edit_own_post",
		"feature_posts", "moderate_post", "read_post",
		// Comment-related
		"create_comment", "delete_any_comment", "delete_own_comment", "edit_any_comment", "edit_own_comment",
		"moderate_comment",
		// User-related
		"ban_user", "create_user", "delete_any_user", "delete_own_profile", "edit_any_user", "edit_own_profile",
		"follow_user", "moderate_user", "read_user_profile", "unfollow_user",
		// Role and permission management
		"assign_roles", "create_roles", "delete_roles", "edit_roles", "manage_roles",
		// Reaction-related
		"create_reaction", "delete_any_reaction", "delete_own_reaction", "edit_any_reaction", "edit_own_reaction",
		"give_reaction",
		// Tag-related
		"create_tag", "delete_tag", "edit_tag", "follow_tag", "moderate_tag", "unfollow_tag",
		// Site-wide and moderation
		"give_suggestion", "manage_analytics", "manage_notifications", "manage_site_settings",
		"need_moderation", "report_content",
	}

	permMap := make(map[string]Permission)
	for _, permName := range allPermissions {
		var perm Permission
		if err := db.WithContext(ctx).Where("name = ?", permName).First(&perm).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				perm = Permission{Name: permName}
				if err := db.WithContext(ctx).FirstOrCreate(&perm).Error; err != nil {
					logger.Error(ctx).Logs(fmt.Sprintf("Failed to create permission: %v, perm: %s", err, permName))
					continue
				}
			} else {
				logger.Error(ctx).Logs(fmt.Sprintf("Database error fetching permission: %v, perm: %s", err, permName))
				continue
			}
		}
		permMap[permName] = perm
	}

	// Define roles with permissions
	roles := []struct {
		Name        string
		Permissions []string
	}{
		{"member", []string{
			"read_post", "create_comment", "edit_own_comment", "delete_own_comment",
			"give_reaction", "follow_tag", "unfollow_tag", "follow_user", "unfollow_user",
			"delete_own_profile", "edit_own_profile", "need_moderation", "report_content",
		}},
		{"author", []string{
			"read_post", "create_post", "edit_own_post", "delete_own_post",
			"create_comment", "edit_own_comment", "delete_own_comment",
			"give_reaction", "follow_tag", "unfollow_tag", "follow_user", "unfollow_user",
			"delete_own_profile", "edit_own_profile", "report_content",
		}},
		{"trusted_member", []string{
			"read_post", "create_post", "edit_own_post", "delete_own_post",
			"create_comment", "edit_own_comment", "delete_own_comment",
			"give_reaction", "follow_tag", "unfollow_tag", "follow_user", "unfollow_user",
			"delete_own_profile", "edit_own_profile", "give_suggestion", "report_content",
		}},
		{"tag_moderator", []string{
			"read_post", "create_post", "edit_own_post", "delete_own_post",
			"create_comment", "edit_own_comment", "delete_own_comment",
			"give_reaction", "follow_tag", "unfollow_tag", "follow_user", "unfollow_user",
			"delete_own_profile", "edit_own_profile", "give_suggestion",
			"moderate_tag", "feature_posts", "ban_user", "report_content",
		}},
		{"moderator", []string{
			"read_post", "create_post", "edit_own_post", "delete_own_post", "edit_any_post",
			"delete_any_post", "moderate_post", "create_comment", "edit_own_comment",
			"delete_own_comment", "edit_any_comment", "moderate_comment", "create_user",
			"edit_user", "delete_user", "moderate_user", "ban_user", "follow_user",
			"unfollow_user", "create_roles", "edit_roles", "delete_roles", "create_reaction",
			"edit_reaction", "delete_reaction", "give_reaction", "create_tag", "edit_tag",
			"delete_tag", "moderate_tag", "follow_tag", "unfollow_tag", "give_suggestion",
			"feature_posts", "report_content",
		}},
		{"admin", allPermissions},
	}

	// Seed roles and permissions into the database
	for _, r := range roles {
		var role Role
		if err := db.WithContext(ctx).Where("name = ?", r.Name).FirstOrCreate(&role).Error; err != nil {
			logger.Error(ctx).Logs(fmt.Sprintf("Failed to create role: %v, role: %s", err, r.Name))
			continue
		}

		// Fetch or link permissions
		var permsToLink []Permission
		for _, permName := range r.Permissions {
			var perm Permission
			if err := db.WithContext(ctx).Where("name = ?", permName).First(&perm).Error; err != nil {
				logger.Error(ctx).Logs(fmt.Sprintf("Permission not found: %v, perm: %s, role: %s", err, permName, r.Name))
				continue
			}
			permsToLink = append(permsToLink, perm)
		}

		// Replace existing permissions with the defined set
		if err := db.WithContext(ctx).Model(&role).Association("Permissions").Replace(permsToLink); err != nil {
			logger.Error(ctx).Logs(fmt.Sprintf("Failed to associate permissions with role: %v, role: %s", err, r.Name))
			continue
		}

		// Fetch permission names for caching
		var permNames []string
		var linkedPerms []Permission
		if err := db.WithContext(ctx).Model(&role).Association("Permissions").Find(&linkedPerms); err != nil {
			logger.Error(ctx).Logs(fmt.Sprintf("Failed to fetch permissions for caching: %v, role: %s", err, r.Name))
			continue
		}
		for _, p := range linkedPerms {
			permNames = append(permNames, p.Name)
		}

		// Cache in Redis
		permsJSON, _ := json.Marshal(permNames)
		if err := redisClient.Set(ctx, "perms:role:"+role.ID.String(), permsJSON, 10*time.Minute).Err(); err != nil {
			logger.Warn(ctx).Logs(fmt.Sprintf("Failed to cache permissions in Redis: %v, role: %s", err, r.Name))
		}
	}

	return nil
}
