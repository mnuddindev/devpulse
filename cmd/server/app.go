package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	routes "github.com/mnuddindev/devpulse/internal/api"
	"github.com/mnuddindev/devpulse/internal/config"
	"github.com/mnuddindev/devpulse/internal/db"
	"github.com/mnuddindev/devpulse/pkg/logger"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"github.com/mnuddindev/devpulse/pkg/utils"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.LoadConfig()

	log, err := logger.NewLogger(ctx)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer func() {
		if err != nil {
			log.Close()
		}
	}()

	rclient, err := storage.NewRedis(ctx, cfg.RedisAddr, "")
	if err != nil {
		log.Error(ctx).WithMeta(utils.Map{"error": err.Error()}).Logs("Failed to initialize Redis")
		panic(err)
	}
	defer func() {
		if err != nil {
			rclient.Close(log)
		}
	}()

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC", cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
	DB, err := db.NewDB(ctx, dsn)
	if err != nil {
		log.Error(ctx).WithMeta(utils.Map{"error": err.Error()}).Logs("Failed to initialize PostgreSQL database")
		panic("DB init failed")
	}
	defer func() {
		if err != nil {
			db.CloseDB()
		}
	}()

	app := fiber.New()

	routes.NewRoutes(ctx, app, cfg, DB, log, rclient)

	go func() {
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
		<-signalChannel
		log.Info(ctx).Logs("Shutting down server...")
		cancel()
		app.Shutdown()
	}()

	log.Info(ctx).Logs("Starting server on :3000")
	if err := app.Listen(cfg.ServerAddr); err != nil {
		log.Error(ctx).WithMeta(utils.Map{"error": err.Error()}).Logs("Server failed")
		os.Exit(1)
	}
}
