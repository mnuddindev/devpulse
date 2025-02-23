package postservices

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
)

// CreateBookmark creates a new bookmark in the database.
func (ps *PostSystem) CreateBookmark(bookmark *models.Bookmark) (*models.Bookmark, error) {
	err := ps.Crud.Create(bookmark)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error creating bookmark")
		return nil, errors.New("error creating bookmark")
	}

	return bookmark, nil
}

// GetBookmarkBy retrieves a bookmark by a given condition.
func (ps *PostSystem) GetBookmarkBy(condition string, args ...interface{}) (*models.Bookmark, error) {
	// an empty instance of bookmark model
	var bookmark models.Bookmark

	// getting bookmark details by given condition
	if err := ps.Crud.GetByCondition(&bookmark, condition, args, []string{"User", "Post", "Collection"}, "", 0, 0); err != nil {
		// log if failed to fetch by condition
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to fetch bookmark by Condition")
		return nil, errors.New("bookmark not found!!")
	}

	// log if successfully fetched the use data by condition
	logger.Log.WithFields(logrus.Fields{
		"bookmark": bookmark,
	}).Info("Bookmark Fetched Successfully!!")

	// return the bookmark data and error
	return &bookmark, nil
}

// UpdateBookmark updates a bookmark in the database.
func (ps *PostSystem) UpdateBookmark(bookmark *models.Bookmark) (*models.Bookmark, error) {
	err := ps.Crud.Update(&bookmark, "id = ?", []interface{}{bookmark.ID}, bookmark)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating bookmark")
		return nil, errors.New("error updating bookmark")
	}

	// return all field with preload using getconditionby
	if err := ps.Crud.GetByCondition(bookmark, "id = ?", []interface{}{bookmark.ID}, []string{"User", "Post", "Collection"}, "", 0, 0); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch bookmarke by Condition")
		return nil, errors.New("bookmarke not found")
	}

	return bookmark, nil
}

// DeleteBookmark deletes a bookmark from the database.
func (ps *PostSystem) DeleteBookmark(id string) error {
	bookmark := &models.Bookmark{}
	err := ps.Crud.Delete(bookmark, "id = ?", []interface{}{id})
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error deleting bookmark")
		return errors.New("error deleting bookmark")
	}
	return nil
}

// UpdateBookmarkMany updates a many-to-many field in the database.
func (ps *PostSystem) UpdateBookmarkMany(bookmarkid uuid.UUID, field string, values []interface{}) error {
	err := ps.Crud.UpdateManyToMany(&models.Bookmark{ID: bookmarkid}, field, values)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating bookmark many")
		return errors.New("error updating bookmark many")
	}
	return nil
}

// GetBookmarks retrieves all bookmarks from the database.
func (ps *PostSystem) GetBookmarks() ([]models.Bookmark, error) {
	// an empty slice of bookmarks
	var bookmarks []models.Bookmark

	// getting all bookmarks
	if err := ps.Crud.GetAll(&bookmarks, []string{"User", "Post", "Collection"}); err != nil {
		// log if failed to fetch all bookmarks
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch all bookmarks")
		return nil, errors.New("failed to fetch all bookmarks")
	}

	// log if successfully fetched all bookmarks
	logger.Log.WithFields(logrus.Fields{
		"bookmarks": bookmarks,
	}).Info("Bookmarks Fetched Successfully!!")

	// return the bookmarks and error
	return bookmarks, nil
}
