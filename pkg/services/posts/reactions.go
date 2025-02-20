package postservices

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
)

// CreateReaction creates a new react for post.
func (ps *PostSystem) CreateReaction(reaction *models.Reaction) (*models.Reaction, error) {
	err := ps.crud.Create(reaction)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error creating reaction")
		return nil, errors.New("error creating reaction")
	}

	return reaction, nil
}

// GetReactionBy retrieves a reaction by a given condition.
func (ps *PostSystem) GetReactionBy(condition string, args ...interface{}) (*models.Reaction, error) {
	// an empty instance of reaction model
	var reaction models.Reaction

	// getting reaction details by given condition
	if err := ps.crud.GetByCondition(&reaction, condition, args, []string{"User", "Post"}, "", 0, 0); err != nil {
		// log if failed to fetch by condition
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to fetch reaction by Condition")
		return nil, errors.New("reaction not found!!")
	}

	return &reaction, nil
}

// UpdateReaction updates a reaction by a given condition.
func (ps *PostSystem) UpdateReaction(reaction *models.Reaction) (*models.Reaction, error) {
	err := ps.crud.Update(&reaction, "id = ?", []interface{}{reaction.ID}, reaction)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating reaction")
		return nil, errors.New("error updating reaction")
	}

	// return all field with preload using getconditionby
	if err := ps.crud.GetByCondition(reaction, "id = ?", []interface{}{reaction.ID}, []string{"User", "Post"}, "", 0, 0); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch reaction by Condition")
		return nil, errors.New("reaction not found")
	}

	return reaction, nil
}

// DeleteReaction deletes a reaction by a given condition.
func (ps *PostSystem) DeleteReaction(condition string, args ...interface{}) error {
	err := ps.crud.Delete(&models.Reaction{}, condition, args)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to delete reaction")
		return errors.New("failed to delete reaction")
	}

	return nil
}

// GetReactions retrieves all reactions.
func (ps *PostSystem) GetReactions() ([]models.Reaction, error) {
	// an empty slice of reaction model
	var reactions []models.Reaction

	// getting all reactions
	if err := ps.crud.GetAll(&reactions, []string{"User", "Post"}); err != nil {
		// log if failed to fetch all reactions
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch all reactions")
		return nil, errors.New("reactions not found!!")
	}

	return reactions, nil
}

// UpdateReactionsMany updates a many-to-many field in the database.
func (ps *PostSystem) UpdateReactionsMany(reactionid uuid.UUID, field string, value interface{}) error {
	err := ps.crud.UpdateManyToMany(&models.Reaction{ID: reactionid}, field, value)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating reaction")
		return errors.New("error updating reaction")
	}
	return err
}
