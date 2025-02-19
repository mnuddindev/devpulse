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
	"github.com/mnuddindev/devpulse/pkg/services/users"
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
	userService := users.NewUserSystem(system.DB)

	// for guest users
	// home router
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"app name:":    config.App,
			"app version:": config.Version,
			"app author:":  config.Author,
			"message:":     "the page you are looking for is getting ready.Please try again leter.",
		})
	})
	app.Post("/register", system.UserController.Registration)
	app.Post("/user/:userid/activate", system.UserController.ActiveUser)
	app.Post("/login", system.UserController.Login)

	// for the guests
	app.Get("/users/profile/id/:userid", system.UserController.UserByID)

	app.Post("/logout", auth.RoleAuth("all"), system.UserController.Logout)

	authgroup := app.Group("/", auth.RefreshTokenMiddleware(userService), auth.IsAuth(userService))
	user := authgroup.Group("/user")
	user.Get("/profile", auth.IsAuth(userService), auth.RoleAuth("all"), system.UserController.UserProfile)
	user.Put("/update/profile/me", auth.IsAuth(userService), auth.RoleAuth("all"), system.UserController.UpdateUserProfile)
	user.Put("/update/notification/me", auth.IsAuth(userService), auth.RoleAuth("all"), system.UserController.UpdateUserNotificationsPref)
	user.Put("/update/customization/me", auth.IsAuth(userService), auth.RoleAuth("all"), system.UserController.UpdateUserCustomization)
	user.Put("/update/account/me", auth.IsAuth(userService), auth.RoleAuth("all"), system.UserController.UpdateUserAccount)
	user.Delete("/account/delete/me", auth.IsAuth(userService), auth.RoleAuth("all"), system.UserController.DeleteUser)

	// protected routes
	users := authgroup.Group("/users")
	users.Post("/:userid/follow", auth.RoleAuth("all"), system.UserController.FollowUser)
	users.Delete("/:userid/unfollow", auth.RoleAuth("all"), system.UserController.UnfollowUser)
	users.Get("/:userid/followers", auth.RoleAuth("all"), system.UserController.GetAllFollowers)
	users.Get("/:userid/following", auth.RoleAuth("all"), system.UserController.GetAllFollowing)
	users.Put("/account/update/:userid", auth.RoleAuth("admin", "moderator"), system.UserController.UpdateUserByID)
	users.Delete("/account/delete/:userid", auth.RoleAuth("admin"), system.UserController.DeleteUserByID)

	// Protected routes
	app.Get("/protected", auth.IsAuth(userService), func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "This is private data!",
			"user_id": userID,
		})
	})
}
