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
func (tm *TagModerator) AddTagModerator(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return utils.NewError(utils.ErrBadRequest.Code, "No user IDs provided")
	}

	tx := db.WithContext(ctx).Begin()
	tag, err := GetTagBy(ctx, rclient, db, "id = ?", []interface{}{tagID})
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
			_, err := user.GetUserBy(ctx, rclient, tx, "id = ?", []interface{}{userID})
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
	rclient.Set(ctx, "tag:"+tag.ID.String(), tagData, 24*time.Hour)
	rclient.Set(ctx, "tag:slug:"+tag.Slug, tagData, 24*time.Hour)
	rclient.Set(ctx, "tag:moderators_count:"+tag.ID.String(), int(currentCount)+len(newModerators), 24*time.Hour)
	rclient.Del(ctx, "tag:moderators:"+tag.ID.String())

	return nil
}

// RemoveTagModerator removes a tag moderator from the database
func (tm *TagModerator) RemoveTagModerator(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return utils.NewError(utils.ErrBadRequest.Code, "No user IDs provided")
	}

	tx := db.WithContext(ctx).Begin()

	tag, err := GetTagBy(ctx, rclient, db, "id = ?", []interface{}{tagID})
	if err != nil {
		return err
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		var currentCount int64
		if err := tx.Model(&TagModerator{}).Where("tag_id = ?", tagID).Count(&currentCount).Error; err != nil {
			return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to count moderators")
		}

		result := tx.Where("tag_id = ? AND user_id IN ?", tagID, userIDs).Delete(&TagModerator{})
		if result.Error != nil {
			return utils.WrapError(result.Error, utils.ErrInternalServerError.Code, "Failed to remove tag moderators")
		}

		countRemoved := int(result.RowsAffected)
		if countRemoved == 0 {
			return nil
		}

		return nil
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	// Update caches
	tagData, _ := json.Marshal(tag)
	rclient.Set(ctx, "tag:"+tag.ID.String(), tagData, 24*time.Hour)
	rclient.Set(ctx, "tag:slug:"+tag.Slug, tagData, 24*time.Hour)
	rclient.Set(ctx, "tag:moderators_count:"+tag.ID.String(), 0, 24*time.Hour)
	rclient.Del(ctx, "tag:moderators:"+tag.ID.String())

	return nil
}

// GetTagModerators retrieves the moderators for a tag
func (tm *TagModerator) GetTagModerators(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID uuid.UUID, page, limit int) ([]user.User, error) {
	if page < 1 || limit < 1 {
		return nil, utils.NewError(utils.ErrBadRequest.Code, "Invalid page or limit")
	}

	cacheKey := fmt.Sprintf("tag:moderators:%s:page:%d:limit:%d", tagID.String(), page, limit)
	if cached, err := rclient.Get(ctx, cacheKey).Result(); err == nil {
		var moderators []user.User
		if json.Unmarshal([]byte(cached), &moderators) == nil {
			return moderators, nil
		}
	}

	var moderators []user.User
	offset := (page - 1) * limit
	err := db.WithContext(ctx).
		Joins("JOIN tag_moderators ON tag_moderators.user_id = users.id").
		Where("tag_moderators.tag_id = ?", tagID).
		Offset(offset).
		Limit(limit).
		Order("tag_moderators.created_at DESC").
		Find(&moderators).Error
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch tag moderators")
	}

	moderatorsData, _ := json.Marshal(moderators)
	rclient.Set(ctx, cacheKey, moderatorsData, 1*time.Hour)

	return moderators, nil
}

// IsModerator checks if a user is a moderator of a tag
func (tm *TagModerator) IsModerator(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID, userID uuid.UUID) (bool, error) {
	cacheKey := fmt.Sprintf("tag:moderator:%s:%s", tagID.String(), userID.String())
	if cached, err := rclient.Get(ctx, cacheKey).Result(); err == nil {
		return cached == "true", nil
	}

	var moderator TagModerator
	err := db.WithContext(ctx).
		Where("tag_id = ? AND user_id = ?", tagID, userID).
		First(&moderator).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			rclient.Set(ctx, cacheKey, "false", 1*time.Hour)
			return false, nil
		}
		return false, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check moderator status")
	}

	rclient.Set(ctx, cacheKey, "true", 1*time.Hour)
	return true, nil
}

// DeleteTagModerators deletes all moderators for a tag
func (tm *TagModerator) DeleteTagModerators(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID uuid.UUID) error {
	tx := db.WithContext(ctx)

	var moderatorCount int64
	if err := tx.Model(&TagModerator{}).Where("tag_id = ?", tagID).Count(&moderatorCount).Error; err != nil {
		return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to count tag moderators")
	}
	if moderatorCount == 0 {
		return nil
	}

	err := tx.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("tag_id = ?", tagID).Delete(&TagModerator{})
		if result.Error != nil {
			return utils.WrapError(result.Error, utils.ErrInternalServerError.Code, "Failed to delete tag moderators")
		}

		return nil
	})
	if err != nil {
		return err
	}

	tx.Commit()

	rclient.Del(ctx, "tag:moderators:"+tagID.String(), "tag:moderators_count:"+tagID.String())

	return nil
}
