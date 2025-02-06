package controllers

import (
	"github.com/mnuddindev/devpulse/config"
	"github.com/mnuddindev/devpulse/gorm"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/services"
	grm "gorm.io/gorm"
)

type CentralSystem struct {
	DB             *grm.DB
	Usercontroller *UserController
}

func StartServices(config *config.Postgres) (*CentralSystem, error) {
	logger.Log.Info("Initializing application system")
	db := gorm.Connect(config)
	userSystem := services.NewUserSystem(db)
	userController := NewUserController(userSystem)
	return &CentralSystem{
		DB:             db,
		Usercontroller: userController,
	}, nil
}
