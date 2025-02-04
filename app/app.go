package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/config"
	"github.com/mnuddindev/devpulse/gorm"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/routes"
	"github.com/sirupsen/logrus"
)

func main() {
	logger.Log.Info("Starting the application...")
	config, err := config.LoadConfig()
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Error while loading config")
		return
	}
	db := gorm.Connect(&config.Postgres)
	app := fiber.New()
	routes.NewRoutes(app, &config.ServerConfig, db)
	app.Listen(config.ServerConfig.Port)
}
