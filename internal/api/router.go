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

func NewRoutes(ctx context.Context, app *fiber.App, cfg *config.Config, db *gorm.DB) {
	log, err := logger.NewLogger(ctx)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer func() {
		if err != nil {
			log.Close()
		}
	}()

	redisClient, err := storage.NewRedis(ctx, cfg.RedisAddr, "")
	if err != nil {
		log.Error(ctx).WithMeta(map[string]string{"error": err.Error()}).Logs("Failed to initialize Redis")
		panic(err)
	}
	defer func() {
		if err != nil {
			redisClient.Close(log)
		}
	}()

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
		redisClient.Close(log)
		log.Close()
	}()
}
