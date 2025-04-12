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

type TagFollower struct {
	TagID     uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"tag_id" validate:"required"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"user_id" validate:"required"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	Tag  Tag       `gorm:"foreignKey:TagID" json:"tag" validate:"-"`
	User user.User `gorm:"foreignKey:UserID" json:"user" validate:"-"`
}

// FollowTag adds users to follow a tag
func FollowTag(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID uuid.UUID, userIDs []uuid.UUID) error {
	tx := db.WithContext(ctx).Begin()

	tag, err := GetTagBy(ctx, rclient, db, "id = ?", []interface{}{tagID})
	if err != nil {
		return err
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		newFollowers := make([]TagFollower, 0, len(userIDs))

		for _, userID := range userIDs {
			_, err := user.GetUserBy(ctx, rclient, tx, "id = ?", []interface{}{userID})
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return utils.NewError(utils.ErrNotFound.Code, "User not found: "+userID.String())
				}
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch user")
			}

			var existing TagFollower
			if err := tx.Where("tag_id = ? AND user_id = ?", tagID, userID).First(&existing).Error; err == nil {
				continue
			} else if err != gorm.ErrRecordNotFound {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check follower status")
			}

			newFollowers = append(newFollowers, TagFollower{
				TagID:  tagID,
				UserID: userID,
			})
		}

		if len(newFollowers) > 0 {
			if err := tx.Create(&newFollowers).Error; err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to add tag followers")
			}

			tag.FollowersCount += len(newFollowers)
			if err := tx.Model(&Tag{ID: tagID}).Update("followers_count", tag.FollowersCount).Error; err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update followers count")
			}

			// Sync TagAnalytics
			if err := SyncTagAnalytics(ctx, rclient, tx, tag.ID, 0, len(newFollowers)); err != nil {
				return err
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
	rclient.Del(ctx, "tag:followers:"+tag.ID.String())

	return nil
}

// UnfollowTag removes users from following a tag
func UnfollowTag(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID uuid.UUID, userIDs []uuid.UUID) error {
	tx := db.WithContext(ctx).Begin()

	tag, err := GetTagBy(ctx, rclient, db, "id = ?", []interface{}{tagID})
	if err != nil {
		return err
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("tag_id = ? AND user_id IN ?", tagID, userIDs).Delete(&TagFollower{})
		if result.Error != nil {
			return utils.WrapError(result.Error, utils.ErrInternalServerError.Code, "Failed to remove tag followers")
		}

		countRemoved := int(result.RowsAffected)
		if countRemoved > 0 {
			tag.FollowersCount -= countRemoved
			if tag.FollowersCount < 0 {
				tag.FollowersCount = 0
			}
			if err := tx.Model(&Tag{ID: tagID}).Update("followers_count", tag.FollowersCount).Error; err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to update followers count")
			}

			if err := SyncTagAnalytics(ctx, rclient, tx, tag.ID, 0, -countRemoved); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	// Update tag cache
	tagData, _ := json.Marshal(tag)
	rclient.Set(ctx, "tag:"+tag.ID.String(), tagData, 24*time.Hour)
	rclient.Set(ctx, "tag:slug:"+tag.Slug, tagData, 24*time.Hour)
	rclient.Del(ctx, "tag:followers:"+tag.ID.String()) // Invalidate followers cache

	return nil
}

// GetTagFollowers retrieves a paginated list of a tag's followers
func GetTagFollowers(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID uuid.UUID, page, limit int) ([]user.User, error) {
	cacheKey := fmt.Sprintf("tag:followers:%s:page:%d:limit:%d", tagID.String(), page, limit)
	if cached, err := rclient.Get(ctx, cacheKey).Result(); err == nil {
		var followers []user.User
		if json.Unmarshal([]byte(cached), &followers) == nil {
			return followers, nil
		}
	}

	var followers []user.User
	offset := (page - 1) * limit
	err := db.WithContext(ctx).
		Joins("JOIN tag_followers ON tag_followers.user_id = users.id").
		Where("tag_followers.tag_id = ?", tagID).
		Offset(offset).
		Limit(limit).
		Find(&followers).Error
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to fetch tag followers")
	}

	followersData, _ := json.Marshal(followers)
	rclient.Set(ctx, cacheKey, followersData, 1*time.Hour)

	return followers, nil
}

// IsFollowingTag checks if a user is following a tag
func IsFollowingTag(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID, userID uuid.UUID) (bool, error) {
	cacheKey := fmt.Sprintf("tag:follower:%s:%s", tagID.String(), userID.String())
	if cached, err := rclient.Get(ctx, cacheKey).Result(); err == nil {
		return cached == "true", nil
	}

	var follower TagFollower
	err := db.WithContext(ctx).
		Where("tag_id = ? AND user_id = ?", tagID, userID).
		First(&follower).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			rclient.Set(ctx, cacheKey, "false", 1*time.Hour)
			return false, nil
		}
		return false, utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to check follower status")
	}

	rclient.Set(ctx, cacheKey, "true", 1*time.Hour)
	return true, nil
}

// DeleteFollowers deletes all followers of a tag
func DeleteFollowers(ctx context.Context, rclient *storage.RedisClient, db *gorm.DB, tagID uuid.UUID) error {
	tx := db.WithContext(ctx)

	tag, err := GetTagBy(ctx, rclient, tx, "id = ?", []interface{}{tagID})
	if err != nil {
		return err
	}

	followerCount := tag.FollowersCount
	err = tx.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("tag_id = ?", tagID).Delete(&TagFollower{})
		if result.Error != nil {
			return utils.WrapError(result.Error, utils.ErrInternalServerError.Code, "Failed to delete tag followers")
		}

		if followerCount > 0 {
			if err := tx.Model(&Tag{ID: tagID}).Update("followers_count", 0).Error; err != nil {
				return utils.WrapError(err, utils.ErrInternalServerError.Code, "Failed to reset followers count")
			}

			if err := SyncTagAnalytics(ctx, rclient, tx, tagID, 0, -followerCount); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	rclient.Del(ctx, "tag:followers:"+tagID.String())

	return nil
}
