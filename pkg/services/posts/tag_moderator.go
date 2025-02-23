package postservices

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
)

// AssignModerator assigns a user as a moderator for a tag. using ManyToMany function
func (ps *PostSystem) AssignModerator(tagID uuid.UUID, user models.User) error {
	// get the tag by tagID
	tag, err := ps.GetTagBy("id = ?", tagID)
	if err != nil {
		return err
	}

	// assign the user as a moderator for the tag
	err = ps.UpdateTagsMany(tag.ID, "Moderators", user)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error assigning moderator")
		return errors.New("error assigning moderator")
	}

	return nil
}

// IsModerator checks if a user is a moderator for a tag.
func (ps *PostSystem) IsModerator(tagID uuid.UUID, userid uuid.UUID) (bool, error) {
	// get the tag by tagID
	tag, err := ps.GetTagBy("tag_id = ? AND user_id = ?", tagID, userid)
	if err != nil {
		return false, err
	}

	// check if the user is a moderator for the tag
	if tag != nil {
		return true, nil
	}

	return false, nil
}

// GetUserModeratedTags retrieves all tags moderated by a user.
func (ps *PostSystem) GetUserModeratedTags(userid uuid.UUID) ([]models.Tag, error) {
	// get all tags moderated by the user
	var tags []models.Tag
	if err := ps.Crud.GetByCondition(&tags, "moderators.id = ?", []interface{}{userid}, []string{"Posts", "Followers", "Moderators", "Analytics"}, "", 0, 0); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error getting moderated tags")
		return nil, errors.New("error getting moderated tags")
	}

	return tags, nil
}

// GetModerators retrieves all moderators for a tag.
func (ps *PostSystem) GetModerators(tagID uuid.UUID) ([]models.User, error) {
	// get the tag by tagID
	tag, err := ps.GetTagBy("id = ?", tagID)
	if err != nil {
		return nil, err
	}

	// return the moderators for the tag
	return tag.Moderators, nil
}

// UpdateTagModerator updates the moderators for a tag.
func (ps *PostSystem) UpdateTagModerator(tagID uuid.UUID, moderators []models.User) error {
	// get the tag by tagID
	tag, err := ps.GetTagBy("id = ?", tagID)
	if err != nil {
		return err
	}

	// update the moderators for the tag
	err = ps.UpdateTagsMany(tag.ID, "Moderators", moderators)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating tag moderators")
		return errors.New("error updating tag moderators")
	}

	return nil
}

// UnassignModerator unassigns a user as a moderator for a tag. using ManyToMany function
func (ps *PostSystem) UnassignModerator(tagID uuid.UUID, user models.User) error {
	// get the tag by tagID
	tag, err := ps.GetTagBy("id = ?", tagID)
	if err != nil {
		return err
	}

	// unassign the user as a moderator for the tag
	err = ps.DeleteTagsMany(tag.ID, "Moderators", user)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error unassigning moderator")
		return errors.New("error unassigning moderator")
	}

	return nil
}
