package controllers

import (
	"bytes"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
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

type UserController struct {
	userSystem *services.UserSystem
}

func NewUserController(userSystem *services.UserSystem) *UserController {
	return &UserController{
		userSystem: userSystem,
	}
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

func StrictBodyParser(c *fiber.Ctx, out interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields() // Reject unknown fields
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}
