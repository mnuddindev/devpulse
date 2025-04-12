package models

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"gorm.io/gorm"
)

type TagModerator struct {
	TagID     uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"tag_id" validate:"required"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"user_id" validate:"required"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	Tag  Tag       `gorm:"foreignKey:TagID" json:"tag" validate:"-"`
	User user.User `gorm:"foreignKey:UserID" json:"user" validate:"-"`
}

const (
	MaxModeratorsPerTag = 50
)

// AddTagModerator adds a new tag moderator to the database
func (tm *TagModerator) AddTagModerator(ctx context.Context, redisClient *storage.RedisClient, gormDB *gorm.DB, tagID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return utils.NewError(utils.ErrBadRequest.Code, "No user IDs provided")
	}

	tx := gormDB.WithContext(ctx).Begin()
	tag, err := GetTagBy(ctx, redisClient, gormDB, "id = ?", []interface{}{tagID})
	if err != nil {
		return err
	}
	if !tag.IsModerated {
		return utils.NewError(utils.ErrBadRequest.Code, "Tag must be moderated to add moderators")
	}

	var currentCount int64
	if err := tx.Model(&TagModerator{}).Where("tag_id = ?", tagID).Count(&currentCount).Error; err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to count moderators")
	}
	if int(currentCount)+len(userIDs) > MaxModeratorsPerTag {
		return utils.NewError(utils.ErrBadRequest.Code, fmt.Sprintf("Tag cannot have more than %d moderators", MaxModeratorsPerTag))
	}

	newModerators := make([]TagModerator, 0, len(userIDs))
	err = tx.Transaction(func(tx *gorm.DB) error {
		var existing []TagModerator
		if err := tx.Where("tag_id = ? AND user_id IN ?", tagID, userIDs).Find(&existing).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check existing moderators")
		}
		existingMap := make(map[uuid.UUID]bool)
		for _, m := range existing {
			existingMap[m.UserID] = true
		}

		for _, userID := range userIDs {
			if existingMap[userID] {
				continue
			}
			_, err := user.GetUserBy(ctx, redisClient, tx, "id = ?", []interface{}{userID})
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return utils.NewError(utils.ErrNotFound.Code, "User not found: "+userID.String())
				}
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch user")
			}

			newModerators = append(newModerators, TagModerator{
				TagID:  tagID,
				UserID: userID,
			})
		}
		if len(newModerators) > 0 {
			if err := tx.Create(&newModerators).Error; err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to add tag moderators")
			}
		}
		return nil
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	tagData, _ := json.Marshal(tag)
	redisClient.Set(ctx, "tag:"+tag.ID.String(), tagData, 24*time.Hour)
	redisClient.Set(ctx, "tag:slug:"+tag.Slug, tagData, 24*time.Hour)
	redisClient.Set(ctx, "tag:moderators_count:"+tag.ID.String(), int(currentCount)+len(newModerators), 24*time.Hour)
	redisClient.Del(ctx, "tag:moderators:"+tag.ID.String())

	return nil
}
