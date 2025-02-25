package main

import (
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/config"
	"github.com/mnuddindev/devpulse/controllers"
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
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:1210",
	})
	system, err := controllers.StartServices(&config.Postgres, client)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed to initialize application systems")
		return
	}
	app := fiber.New()
	routes.NewRoutes(app, &config.ServerConfig, system, client)
	app.Listen(config.ServerConfig.Port)
}
