package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/mnuddindev/devpulse/config"
	"github.com/mnuddindev/devpulse/controllers"
)

func NewRoutes(app *fiber.App, config *config.ServerConfig, system *controllers.CentralSystem) {
	app.Use(
		logger.New(),
		recover.New(),
		cors.New(cors.Config{
			AllowOrigins:     "http://localhost:3000", // Specify your frontend origin
			AllowCredentials: true,
			AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		}),
		compress.New(compress.Config{
			Level: compress.LevelBestCompression,
		}),
	)

	// home router
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"app name:":    config.App,
			"app version:": config.Version,
			"app author:":  config.Author,
			"message:":     "the page you are looking for is getting ready.Please try again leter.",
		})
	})
	app.Post("/register", system.Usercontroller.Registration)
}
