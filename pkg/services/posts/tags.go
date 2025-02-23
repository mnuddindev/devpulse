package postservices

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
)

// CreatesTag creates a new Tag in the database.
func (ps *PostSystem) CreatesTag(tag *models.Tag) (*models.Tag, error) {
	err := ps.Crud.Create(tag)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error creating tag")
		return nil, errors.New("error creating tag")
	}

	return tag, nil
}

// GetTagBy retrieves a tag by a given condition.
func (ps *PostSystem) GetTagBy(condition string, args ...interface{}) (*models.Tag, error) {
	// an empty instance of tag model
	var tag models.Tag

	// getting tag details by given condition
	if err := ps.Crud.GetByCondition(&tag, condition, args, []string{"Posts", "Follower", "Moderator", "Analytics"}, "", 0, 0); err != nil {
		// log if failed to fetch by condition
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to fetch tag by Condition")
		return nil, errors.New("tag not found!!")
	}

	// log if successfully fetched the use data by condition
	logger.Log.WithFields(logrus.Fields{
		"tag": tag,
	}).Info("Tag Fetched Successfully!!")

	// return the tag data and error
	return &tag, nil
}

// UpdateTag updates a tag in the database.
func (ps *PostSystem) UpdateTag(tag *models.Tag) (*models.Tag, error) {
	// update the tag in the database
	err := ps.Crud.Update(&tag, "id = ?", []interface{}{tag.ID}, tag)
	if err != nil {
		// log if failed to update the tag
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating tag")
		return nil, errors.New("error updating tag")
	}

	// reload the tag to get the details using preload
	err = ps.Crud.GetByCondition(&tag, "id = ?", []interface{}{tag.ID}, []string{"Posts", "Follower", "Moderator", "Analytics"}, "", 0, 0)
	if err != nil {
		// log if failed to fetch the tag
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error fetching full tag")
	}

	return tag, nil
}

// DeleteTag deletes a tag from the database.
func (ps *PostSystem) DeleteTag(tag *models.Tag) error {
	// delete the tag from the database
	err := ps.Crud.Delete(&tag, "id = ?", []interface{}{tag.ID})
	if err != nil {
		// log if failed to delete the tag
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error deleting tag")
		return errors.New("error deleting tag")
	}

	return nil
}

// GetTags retrieves all tags from the database.
func (ps *PostSystem) GetTags() ([]models.Tag, error) {
	// an empty slice of tags
	var tags []models.Tag

	// get all tags from the database
	if err := ps.Crud.GetAll(&tags, []string{"Posts", "Follower", "Moderator", "Analytics"}); err != nil {
		// log if failed to fetch all tags
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error fetching all tags")
		return nil, errors.New("error fetching all tags")
	}

	// log if successfully fetched all tags
	logger.Log.WithFields(logrus.Fields{
		"tags": tags,
	}).Info("Tags Fetched Successfully!!")

	return tags, nil
}

// UpdateTagsMany updates many-tp-many relationship of tags with posts.
func (ps *PostSystem) UpdateTagsMany(tagid uuid.UUID, assoc string, value interface{}) error {
	// update many-to-many relationship of tags with posts
	err := ps.Crud.UpdateManyToMany(&models.Tag{ID: tagid}, assoc, value)
	if err != nil {
		// log if failed to update many-to-many relationship
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating tag relations")
		return errors.New("error updating tag relations")
	}

	return nil
}

// DeleteTagsMany deletes many-to-many relationship of tags with posts.
func (ps *PostSystem) DeleteTagsMany(tagid uuid.UUID, assoc string, value interface{}) error {
	// delete many-to-many relationship of tags with posts
	err := ps.Crud.DeleteManyToMany(&models.Tag{ID: tagid}, assoc, value)
	if err != nil {
		// log if failed to delete many-to-many relationship
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error deleting tag relations")
		return errors.New("error deleting tag relations")
	}

	return nil
}
