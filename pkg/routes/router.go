package routes

import (
	"time"

	"github.com/go-redis/redis/v8"
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

func NewRoutes(app *fiber.App, config *config.ServerConfig, system *controllers.CentralSystem, client *redis.Client) {
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
	userService := users.NewUserSystem(system.DB, client)

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

	app.Post("/logout", auth.PermissionAuth(system.DB, "read_post"), system.UserController.Logout)

	authgroup := app.Group("/", auth.RefreshTokenMiddleware(userService, system.DB, client))
	user := authgroup.Group("/user")
	user.Get("/profile", auth.PermissionAuth(system.DB, "read_post"), system.UserController.UserProfile)
	user.Put("/update/profile/me", auth.PermissionAuth(system.DB, "edit_own_profile"), system.UserController.UpdateUserProfile)
	user.Put("/update/notification/me", auth.PermissionAuth(system.DB, "edit_own_profile"), system.UserController.UpdateUserNotificationsPref)
	// Update customization (user-specific permission)
	user.Put("/update/customization/me", auth.PermissionAuth(system.DB, "edit_own_profile"), system.UserController.UpdateUserCustomization)
	// Update account details (user-specific permission)
	user.Put("/update/account/me", auth.PermissionAuth(system.DB, "edit_own_profile"), system.UserController.UpdateUserAccount)
	// Delete own account (user-specific permission)
	user.Delete("/account/delete/me", auth.PermissionAuth(system.DB, "delete_own_account"), system.UserController.DeleteUserByID)

	// protected routes
	users := authgroup.Group("/users")
	users.Post("/:userid/follow", auth.PermissionAuth(system.DB, "create_post"), system.UserController.FollowUser)
	// Unfollow a user (any authenticated user)
	users.Delete("/:userid/unfollow", auth.PermissionAuth(system.DB, "create_post"), system.UserController.UnfollowUser)
	// Get followers of a user (any authenticated user)
	users.Get("/:userid/followers", auth.PermissionAuth(system.DB, "read_post"), system.UserController.GetAllFollowers)
	// Get following list of a user (any authenticated user)
	users.Get("/:userid/following", auth.PermissionAuth(system.DB, "read_post"), system.UserController.GetAllFollowing)
	// Update another user’s account (admin or moderator)
	users.Put("/account/update/:userid", auth.PermissionAuth(system.DB, "manage_users", "moderate_content"), system.UserController.UpdateUserByID)
	// Delete another user’s account (admin only)
	users.Delete("/account/delete/:userid", auth.PermissionAuth(system.DB, "admin"), system.UserController.DeleteUserByID)

	postgroup := authgroup.Group("/posts")
	postgroup.Post("/", auth.PermissionAuth(system.DB, "create_post"), system.PostController.NewPost)

	// Protected routes
	app.Get("/protected", func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "This is private data!",
			"user_id": userID,
		})
	})
}
