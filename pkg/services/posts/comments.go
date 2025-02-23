package postservices

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
)

// CreateComment creates a new comment in the database.
func (ps *PostSystem) CreateComment(comment *models.Comment) (*models.Comment, error) {
	err := ps.Crud.Create(comment)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error creating comment")
		return nil, errors.New("error creating comment")
	}

	return comment, nil
}

// GetCommentBy retrieves a comment by a given condition.
func (ps *PostSystem) GetCommentBy(condition string, args ...interface{}) (*models.Comment, error) {
	// an empty instance of comment model
	var comment models.Comment

	// getting comment details by given condition
	if err := ps.Crud.GetByCondition(&comment, condition, args, []string{"Author", "Post", "ParentComment", "Replies", "Mentions", "Reactions", "Flags"}, "", 0, 0); err != nil {
		// log if failed to fetch by condition
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to fetch comment by Condition")
		return nil, errors.New("comment not found!!")
	}

	// log if successfully fetched the use data by condition
	logger.Log.WithFields(logrus.Fields{
		"comment": comment,
	}).Info("User Fetched Successfully!!")

	// return the comment data and error
	return &comment, nil
}

// UpdateComment updates a comment in the database.
func (ps *PostSystem) UpdateComment(comment *models.Comment) (*models.Comment, error) {
	err := ps.Crud.Update(&comment, "id = ?", []interface{}{comment.ID}, comment)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating comment")
		return nil, errors.New("error updating comment")
	}

	// return all field with preload using getconditionby
	err = ps.Crud.GetByCondition(comment, "id = ?", []interface{}{comment.ID}, []string{"Author", "Post", "ParentComment", "Replies", "Mentions", "Reactions", "Flags"}, "", 0, 0)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error fetching full comment")
	}

	return comment, nil
}

// DeleteComment deletes a comment from the database.
func (ps *PostSystem) DeleteComment(id string) error {
	comment := &models.Comment{}
	err := ps.Crud.Delete(comment, "id = ?", []interface{}{id})
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error deleting comment")
		return errors.New("error deleting comment")
	}
	return nil
}

// Comments retrieves all comments from the database.
func (ps *PostSystem) Comments() ([]models.Comment, error) {
	comments := []models.Comment{}
	err := ps.Crud.GetAll(&comments, []string{"Author", "Post", "ParentComment", "Replies", "Mentions", "Reactions", "Flags"})
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error fetching comments")
		return nil, errors.New("error fetching comments")
	}
	return comments, nil
}

// UpdateCommentMany updates a many-to-many field in the database.
func (ps *PostSystem) UpdateCommentMany(commentid uuid.UUID, field string, values []interface{}) error {
	err := ps.Crud.UpdateManyToMany(&models.Comment{ID: commentid}, field, values)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating comment many")
		return errors.New("error updating comment many")
	}
	return nil
}
