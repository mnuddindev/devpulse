package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/config"
	"github.com/mnuddindev/devpulse/gorm"
)

func main() {
	config, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error while loading config: %v\n", err)
		return
	}
	db := gorm.Connect(&config.Postgres)
	fmt.Println(db)
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(fmt.Sprintf("%s running on port %s", config.App, config.ServerConfig.Port))
	})
	app.Listen(fmt.Sprintf("%s", config.ServerConfig.Port))
}
