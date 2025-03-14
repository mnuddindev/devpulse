package models

import (
	"context"
	"encoding/json"
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
	var allPermissions []string
	permissions := []Permission{
		// Post-related permissions
		{Name: "create_post"}, {Name: "delete_any_post"}, {Name: "delete_own_post"}, {Name: "edit_any_post"},
		{Name: "edit_own_post"}, {Name: "feature_posts"}, {Name: "moderate_post"},
		{Name: "read_post"},

		// Comment-related permissions
		{Name: "create_comment"}, {Name: "delete_any_comment"}, {Name: "delete_own_comment"},
		{Name: "edit_any_comment"}, {Name: "edit_own_comment"},
		{Name: "moderate_comment"},

		// User-related permissions
		{Name: "ban_user"}, {Name: "create_user"}, {Name: "delete_any_user"}, {Name: "delete_own_profile"},
		{Name: "edit_any_user"}, {Name: "edit_own_profile"}, {Name: "follow_user"},
		{Name: "moderate_user"}, {Name: "read_user_profile"},
		{Name: "unfollow_user"},

		// Role and permission management
		{Name: "assign_roles"}, {Name: "create_roles"}, {Name: "delete_roles"},
		{Name: "edit_roles"}, {Name: "manage_roles"},

		// Reaction-related permissions
		{Name: "create_reaction"}, {Name: "delete_any_reaction"}, {Name: "delete_own_reaction"},
		{Name: "edit_any_reaction"}, {Name: "edit_own_reaction"},

		// Tag-related permissions
		{Name: "create_tag"}, {Name: "delete_tag"}, {Name: "edit_tag"},
		{Name: "follow_tag"}, {Name: "moderate_tag"},
		{Name: "unfollow_tag"},

		// Site-wide and moderation permissions
		{Name: "give_suggestion"}, {Name: "manage_analytics"}, {Name: "manage_notifications"},
		{Name: "manage_site_settings"}, {Name: "need_moderation"},
		{Name: "report_content"},
	}
	for _, perm := range permissions {
		allPermissions = append(allPermissions, perm.Name)
		db.FirstOrCreate(&perm, Permission{Name: perm.Name})
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
		if err := db.WithContext(ctx).Where("name = ?", r.Name).First(&role).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				role = Role{Name: r.Name}
				if err := db.Create(&role).Error; err != nil {
					logger.Error(ctx).WithMeta(utils.Map{
						"error": err.Error(),
						"role":  r.Name,
					}).Logs("Failed to create role")
					continue
				}
			} else {
				logger.Error(ctx).WithMeta(utils.Map{
					"error": err.Error(),
					"role":  r.Name,
				}).Logs("Database error fetching role")
				continue
			}
		}

		// Fetch or create permissions and associate them with the role
		var perms []Permission
		for _, permName := range r.Permissions {
			var perm Permission
			if err := db.WithContext(ctx).Where("name = ?", permName).First(&perm).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					perm = Permission{Name: permName}
					if err := db.Create(&perm).Error; err != nil {
						logger.Error(ctx).WithMeta(utils.Map{
							"error": err.Error(),
							"role":  permName,
						}).Logs("Failed to create permission")
						continue
					}
				} else {
					logger.Error(ctx).WithMeta(utils.Map{
						"error": err.Error(),
						"role":  permName,
					}).Logs("Database error fetching permission")
					continue
				}
			}
			perms = append(perms, perm)
		}

		// Associate the permissions with the role
		if err := db.WithContext(ctx).Model(&role).Association("Permissions").Replace(perms); err != nil {
			logger.Error(ctx).WithMeta(utils.Map{
				"error": err.Error(),
				"role":  r.Name,
			}).Logs("Failed to associate permissions with role")
		}
	}
	return nil
}
