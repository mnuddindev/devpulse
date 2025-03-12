package models

import (
	"context"
	"encoding/json"
	"time"

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

type Permission struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"size:50;not null;unique" json:"name"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// SeedRoles initializes default roles and permissions.
func SeedRoles(ctx context.Context, gormDB *gorm.DB, redisClient *storage.RedisClient) error {
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
		if err := gormDB.WithContext(ctx).Where("name = ?", r.Name).FirstOrCreate(&role).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to seed role: "+r.Name)
		}

		for _, permName := range r.Permissions {
			var perm Permission
			if err := gormDB.WithContext(ctx).Where("name = ?", permName).FirstOrCreate(&perm).Error; err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to seed permission: "+permName)
			}
			gormDB.WithContext(ctx).Model(&role).Association("Permissions").Append(&perm)
		}

		// Cache in Redis
		var perms []string
		gormDB.WithContext(ctx).Model(&role).Association("Permissions").Find(&perms)
		permsJSON, _ := json.Marshal(perms)
		redisClient.Set(ctx, "perms:role:"+r.Name, permsJSON, 10*time.Minute)
	}

	return nil
}
