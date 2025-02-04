package routes

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/mnuddindev/devpulse/config"
	"gorm.io/gorm"
)

func NewRoutes(app *fiber.App, config *config.ServerConfig, db *gorm.DB) {
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
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(fmt.Sprintf("%s running on port %s", config.App, config.Port))
	})
}
