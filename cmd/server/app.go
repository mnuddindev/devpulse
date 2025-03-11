package main

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/internal/config"
)

func main() {
	app := fiber.New()

	_, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cfg := config.LoadConfig()

	app.Listen(cfg.ServerAddr)
}
