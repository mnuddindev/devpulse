package routes

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/mnuddindev/devpulse/internal/config"
	"github.com/mnuddindev/devpulse/pkg/logger"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"gorm.io/gorm"
)

func NewRoutes(ctx context.Context, app *fiber.App, cfg *config.Config, db *gorm.DB, log *logger.Logger, rclient *storage.RedisClient) {
	app.Use(
		logger.SetupLogger(log),
		recover.New(),
		cors.New(
			cors.Config{
				AllowOrigins:     "http://localhost:3000",
				AllowCredentials: true,
				AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
			},
		),
		compress.New(
			compress.Config{
				Level: compress.LevelBestCompression,
			},
		),
		limiter.New(
			limiter.Config{
				Expiration: 1 * time.Minute,
				Max:        10,
				KeyGenerator: func(c *fiber.Ctx) string {
					return c.IP()
				},
			},
		),
	)
	app.Use(log.Middleware())

	go func() {
		<-ctx.Done()
		rclient.Close(log)
		log.Close()
	}()
}
