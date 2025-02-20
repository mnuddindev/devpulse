package postservices

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/gorm"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
	grm "gorm.io/gorm"
)

// PostSystem struct that holds a reference to the CRUD operations using Gorm.
type PostSystem struct {
	crud *gorm.GormDB
}

// NewPostSystem initializes a new UserSystem with a given database connection.
func NewPostSystem(db *grm.DB) *PostSystem {
	return &PostSystem{
		crud: gorm.NewGormDB(db),
	}
}

// CreatePost creates a new post in the database.
func (ps *PostSystem) CreatePost(post *models.Posts) (*models.Posts, error) {
	err := ps.crud.Create(post)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error creating post")
		return nil, errors.New("error creating post")
	}

	return post, nil
}

// GetPostBy retrieves a post by a given condition.
func (ps *PostSystem) GetPostBy(condition string, args ...interface{}) (*models.Posts, error) {
	// an empty instance of post model
	var post models.Posts

	// getting post details by given condition
	if err := ps.crud.GetByCondition(&post, condition, args, []string{"Author", "Series", "Tags", "Comments", "Reactions", "Bookmarks", "Mentions", "CoAuthors"}, "", 0, 0); err != nil {
		// log if failed to fetch by condition
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to fetch post by Condition")
		return nil, errors.New("post not found!!")
	}

	// log if successfully fetched the use data by condition
	logger.Log.WithFields(logrus.Fields{
		"post": post,
	}).Info("User Fetched Successfully!!")

	// return the post data and error
	return &post, nil
}

// UpdatePost updates a post in the database.
func (ps *PostSystem) UpdatePost(post *models.Posts) (*models.Posts, error) {
	err := ps.crud.Update(&post, "id = ?", []interface{}{post.ID}, post)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating post")
		return nil, errors.New("error updating post")
	}

	// return all field with preload using getconditionby
	if err := ps.crud.GetByCondition(post, "id = ?", []interface{}{post.ID}, []string{"Author", "Series", "Tags", "Comments", "Reactions", "Bookmarks", "Mentions", "CoAuthors"}, "", 0, 0); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch post by Condition")
		return nil, errors.New("post not found")
	}

	return post, nil
}

// DeletePost deletes a post from the database.
func (ps *PostSystem) DeletePost(id string) error {
	post := &models.Posts{}
	err := ps.crud.Delete(post, "id = ?", []interface{}{id})
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error deleting post")
		return errors.New("error deleting post")
	}
	return nil
}

// Posts retrieves all posts from the database.
func (ps *PostSystem) Posts() ([]models.Posts, error) {
	posts := []models.Posts{}
	err := ps.crud.GetAll(&posts, []string{"Author", "Series", "Tags", "Comments", "Reactions", "Bookmarks", "Mentions", "CoAuthors"})
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error fetching all posts")
		return nil, errors.New("error fetching all posts")
	}
	return posts, nil
}

// UpdatePostMany updates a many-to-many field in the database.
func (ps *PostSystem) UpdatePostMany(postID uuid.UUID, assoc string, data interface{}) error {
	err := ps.crud.UpdateManyToMany(&models.Posts{ID: postID}, assoc, data)
	if err != nil {
		logger.Log.Error(err)
		return errors.New("error updating many to many")
	}
	return nil
}
