package postservices

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
)

// CreateCollection creates a new collection in the database.
func (ps *PostSystem) CreateCollection(collection *models.Collection) (*models.Collection, error) {
	err := ps.crud.Create(collection)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error creating collection")
		return nil, errors.New("error creating collection")
	}

	return collection, nil
}

// GetCollectionBy retrieves a collection by a given condition.
func (ps *PostSystem) GetCollectionBy(condition string, args ...interface{}) (*models.Collection, error) {
	// an empty instance of collection model
	var collection models.Collection

	// getting collection details by given condition
	if err := ps.crud.GetByCondition(&collection, condition, args, []string{}, "", 0, 0); err != nil {
		// log if failed to fetch by condition
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to fetch collection by Condition")
		return nil, errors.New("collection not found!!")
	}

	// log if successfully fetched the use data by condition
	logger.Log.WithFields(logrus.Fields{
		"collection": collection,
	}).Info("Collection Fetched Successfully!!")

	// return the collection data and error
	return &collection, nil
}

// UpdateCollection updates a collection in the database.
func (ps *PostSystem) UpdateCollection(collection *models.Collection) (*models.Collection, error) {
	err := ps.crud.Update(&collection, "id = ?", []interface{}{collection.ID}, collection)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating collection")
		return nil, errors.New("error updating collection")
	}

	// reload the bookmark to get the user, post and collection details using preload
	err = ps.crud.GetByCondition(&collection, "id = ?", []interface{}{collection.ID}, []string{"User", "Bookmarks"}, "", 0, 0)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error fetching full collection")
	}

	return collection, nil
}

// DeleteCollection deletes a collection from the database.
func (ps *PostSystem) DeleteCollection(id string) error {
	collection := &models.Collection{}
	err := ps.crud.Delete(collection, "id = ?", []interface{}{id})
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error deleting collection")
		return errors.New("error deleting collection")
	}

	// if a collection deleted, all its bookmarks should be deleted too
	err = ps.crud.Delete(&models.Bookmark{}, "collection_id = ?", []interface{}{id})
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error deleting collection bookmarks")
		return errors.New("error deleting collection bookmarks")
	}

	return nil
}

// UpdateCollectionMany updates a many-to-many field in the database.
func (ps *PostSystem) UpdateCollectionMany(collectionid uuid.UUID, field string, values []interface{}) error {
	err := ps.crud.UpdateManyToMany(&models.Collection{ID: collectionid}, field, values)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating collection many")
		return errors.New("error updating collection many")
	}
	return nil
}

// GetCollections retrieves all collections from the database.
func (ps *PostSystem) GetCollections() ([]models.Collection, error) {
	// an empty slice of collections
	var collections []models.Collection

	// getting all collections
	if err := ps.crud.GetAll(&collections, []string{"User", "Bookmarks"}); err != nil {
		// log if failed to fetch all collections
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch all collections")
		return nil, errors.New("failed to fetch all collections")
	}

	// log if successfully fetched all collections
	logger.Log.WithFields(logrus.Fields{
		"collections": collections,
	}).Info("Collections Fetched Successfully!!")

	// return the collections and error
	return collections, nil
}
