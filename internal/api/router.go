package routes

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	v1 "github.com/mnuddindev/devpulse/internal/api/v1"
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

	v1.DB = db
	v1.Redis = rclient
	v1.Logger = log

	app.Post("/register", v1.Register)
	app.Post("/activate", v1.ActivateUser)
	app.Post("/login", v1.Login)
	app.Post("/logout", v1.Logout)
	app.Post("/refresh-token", v1.Refresh)

	go func() {
		<-ctx.Done()
		rclient.Close(log)
		log.Close()
	}()
}
