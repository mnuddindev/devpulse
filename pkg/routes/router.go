package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/mnuddindev/devpulse/config"
	"github.com/mnuddindev/devpulse/controllers"
	"github.com/mnuddindev/devpulse/pkg/auth"
	"github.com/mnuddindev/devpulse/pkg/services"
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
		limiter.New(limiter.Config{
			Expiration: 1 * time.Minute, // Track requests per minute
			Max:        10,              // Allow 10 requests per minute
			KeyGenerator: func(c *fiber.Ctx) string {
				return c.IP() // Rate-limit based on IP
			},
		}),
	)

	// userservice to access crud
	userService := services.NewUserSystem(system.DB)

	// refresh token middleware
	app.Use(auth.RefreshTokenMiddleware(userService))

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
	app.Post("/user/:userid/activate", system.Usercontroller.ActiveUser)
	app.Post("/login", system.Usercontroller.Login)
	app.Post("/logout", system.Usercontroller.Logout)

	user := app.Group("/user", auth.IsAuth())

	user.Get("/profile", auth.RoleAuth("all"), system.Usercontroller.UserProfile)
	user.Put("/update/profile/me", auth.RoleAuth("all"), system.Usercontroller.UpdateUserProfile)
	user.Put("/update/notification/me", auth.RoleAuth("all"), system.Usercontroller.UpdateUserNotificationsPref)
	user.Put("/update/customization/me", auth.RoleAuth("all"), system.Usercontroller.UpdateUserCustomization)
	user.Put("/update/account/me", auth.RoleAuth("all"), system.Usercontroller.UpdateUserAccount)
	user.Delete("/account/delete/me", auth.RoleAuth("admin"), system.Usercontroller.DeleteUser)

	// Protected routes
	app.Get("/protected", auth.IsAuth(), func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "This is private data!",
			"user_id": userID,
		})
	})
}
