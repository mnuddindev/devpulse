package controllers

import (
	"github.com/go-redis/redis/v8"
	"github.com/mnuddindev/devpulse/config"
	postscontroller "github.com/mnuddindev/devpulse/controllers/posts"
	"github.com/mnuddindev/devpulse/controllers/users"
	"github.com/mnuddindev/devpulse/gorm"
	"github.com/mnuddindev/devpulse/pkg/logger"
	postservices "github.com/mnuddindev/devpulse/pkg/services/posts"
	usr "github.com/mnuddindev/devpulse/pkg/services/users"
	grm "gorm.io/gorm"
)

type CentralSystem struct {
	DB             *grm.DB
	UserController *users.UserController
	PostController *postscontroller.PostController
}

func StartServices(config *config.Postgres, client *redis.Client) (*CentralSystem, error) {
	logger.Log.Info("Initializing application system")
	db := gorm.Connect(config)
	userSystem := usr.NewUserSystem(db, client)
	postSystem := postservices.NewPostSystem(db, client)
	userController := users.NewUserController(userSystem, db, client)
	postController := postscontroller.NewPostController(postSystem, db, client)
	return &CentralSystem{
		DB:             db,
		UserController: userController,
		PostController: postController,
	}, nil
}
