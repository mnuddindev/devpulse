package postservices

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
)

// CreateCommentFlag creates a new comment flag in the database.
func (ps *PostSystem) CreateCommentFlag(commentFlag *models.CommentFlag) (*models.CommentFlag, error) {
	err := ps.Crud.Create(commentFlag)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error creating comment flag")
		return nil, errors.New("error creating comment flag")
	}

	return commentFlag, nil
}

// GetCommentFlagBy retrieves a comment flag by a given condition.
func (ps *PostSystem) GetCommentFlagBy(condition string, args ...interface{}) (*models.CommentFlag, error) {
	// an empty instance of comment flag model
	var commentFlag models.CommentFlag

	// getting comment flag details by given condition
	if err := ps.Crud.GetByCondition(&commentFlag, condition, args, []string{"User", "Comment"}, "", 0, 0); err != nil {
		// log if failed to fetch by condition
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to fetch comment flag by Condition")
		return nil, errors.New("comment flag not found!!")
	}

	// log if successfully fetched the use data by condition
	logger.Log.WithFields(logrus.Fields{
		"commentFlag": commentFlag,
	}).Info("Comment Flag Fetched Successfully!!")

	// return the comment flag data and error
	return &commentFlag, nil
}

// UpdateCommentFlag updates an existing comment flag in the database.
func (ps *PostSystem) UpdateCommentFlag(commentFlag *models.CommentFlag) (*models.CommentFlag, error) {
	err := ps.Crud.Update(&commentFlag, "id = ?", []interface{}{commentFlag.ID}, commentFlag)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating comment flag")
		return nil, errors.New("error updating comment flag")
	}

	// return all field with preload using getconditionby
	if err := ps.Crud.GetByCondition(commentFlag, "id = ?", []interface{}{commentFlag.ID}, []string{"User", "Comment"}, "", 0, 0); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch comment flag by Condition")
		return nil, errors.New("comment flag not found!!")
	}

	return commentFlag, nil
}

// DeleteCommentFlag deletes a comment flag from the database.
func (ps *PostSystem) DeleteCommentFlag(flagid uuid.UUID) error {
	err := ps.Crud.Delete(&models.CommentFlag{}, "id = ?", []interface{}{flagid})
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error deleting comment flag")
		return errors.New("error deleting comment flag")
	}

	return nil
}

// GetCommentFlags retrieves all comment flags from the database.
func (ps *PostSystem) GetCommentFlags() ([]models.CommentFlag, error) {
	commentflag := []models.CommentFlag{}
	err := ps.Crud.GetAll(&commentflag, []string{"User", "Comment"})
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error fetching all comment flags")
		return nil, errors.New("error fetching all comment flags")
	}
	return commentflag, nil
}

// UpdateCommentFlagMany updates a many-to-many field in the database.
func (ps *PostSystem) UpdateCommentFlagMany(flagID uuid.UUID, assoc string, data interface{}) error {
	err := ps.Crud.UpdateManyToMany(&models.CommentFlag{ID: flagID}, assoc, data)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating comment flag many")
		return errors.New("error updating comment flag many")
	}
	return nil
}
