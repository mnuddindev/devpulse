package main

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/internal/config"
	"github.com/mnuddindev/devpulse/internal/db"
	"github.com/mnuddindev/devpulse/internal/models"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/utils"
)

func main() {
	ctx := context.Background()

	cfg := config.LoadConfig()

	logger, err := logger.NewLogger(ctx)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Close()

	redisClient, err := db.NewRedis(ctx, cfg.RedisAddr, "")
	if err != nil {
		logger.Error(ctx).WithMeta(utils.Map{"error": err.Error()}).Logs("Failed to initialize Redis")
		panic(err)
	}
	defer redisClient.Close(logger)

	dsn := "host=localhost user=postgres password=secret dbname=blogblaze port=5432 sslmode=disable TimeZone=UTC"
	gormDB, err := db.NewDB(
		ctx,
		dsn,
		[]interface{}{
			&models.User{},
		},
		db.WithLogger(logger),
	)
	if err != nil {
		logger.Error(ctx).WithMeta(utils.Map{"error": err.Error()}).Logs("Failed to initialize PostgreSQL database")
		panic("DB init failed")
	}
	defer db.CloseDB(logger)

	app := fiber.New()

	app.Listen(cfg.ServerAddr)
}
