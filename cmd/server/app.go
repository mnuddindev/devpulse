package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	routes "github.com/mnuddindev/devpulse/internal/api"
	"github.com/mnuddindev/devpulse/internal/config"
	"github.com/mnuddindev/devpulse/internal/db"
	"github.com/mnuddindev/devpulse/pkg/utils"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.LoadConfig()

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC", cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
	DB, err := db.NewDB(ctx, dsn)
	if err != nil {
		log.Fatal(utils.NewError(utils.ErrInternalServerError.Code, "Failed to initialize PostgreSQL database", err.Error()))
		panic("DB init failed")
	}
	defer func() {
		if err != nil {
			db.CloseDB()
		}
	}()

	app := fiber.New()

	routes.NewRoutes(ctx, app, cfg, DB)

	go func() {
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
		<-signalChannel
		log.Fatal("Shutting down server...")
		cancel()
		app.Shutdown()
	}()

	if err := app.Listen(cfg.ServerAddr); err != nil {
		fmt.Println(utils.NewError(utils.ErrInternalServerError.Code, "Server Failed!!", err.Error()))
		os.Exit(1)
	}
}
