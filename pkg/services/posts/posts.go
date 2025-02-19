package postservices

import (
	"errors"

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

func (ps *PostSystem) CreatePost(post *models.Posts) (*models.Posts, error) {
	var posts models.Posts
	if err := ps.crud.GetByCondition(&posts, "slug = ?", []interface{}{post.Slug}, []string{}, "", 0, 0); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"post":  post,
		}).Error("Slug already exists")
		return &models.Posts{}, errors.New("Slug already exists")
	}
	if err := ps.crud.Create(post); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"post":  post,
		}).Error("failed to submitted post")
		return &models.Posts{}, errors.New("failed to submitted post")

	}

	// Log the successful creation of the post.
	logger.Log.WithFields(logrus.Fields{
		"post": post,
	}).Info("Post submitted successfully!!")

	return nil, nil
}
