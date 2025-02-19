package controllers

import (
	"github.com/mnuddindev/devpulse/config"
	"github.com/mnuddindev/devpulse/controllers/users"
	"github.com/mnuddindev/devpulse/gorm"
	"github.com/mnuddindev/devpulse/pkg/logger"
	usr "github.com/mnuddindev/devpulse/pkg/services/users"
	grm "gorm.io/gorm"
)

type CentralSystem struct {
	DB             *grm.DB
	UserController *users.UserController
}

func StartServices(config *config.Postgres) (*CentralSystem, error) {
	logger.Log.Info("Initializing application system")
	db := gorm.Connect(config)
	userSystem := usr.NewUserSystem(db)
	userController := users.NewUserController(userSystem)
	return &CentralSystem{
		DB:             db,
		UserController: userController,
	}, nil
}
