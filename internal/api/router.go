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
	"github.com/mnuddindev/devpulse/internal/auth"
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

	opt := auth.Options{
		DB:      db,
		Rclient: rclient,
		Logger:  log,
	}

	app.Post("/register", v1.Register)
	app.Post("/activate", v1.ActivateUser)
	app.Post("/login", v1.Login)
	app.Post("/logout", v1.Logout)
	app.Post("/refresh-token", v1.Refresh)
	app.Post("/forgot-password", v1.ForgotPassword)
	app.Post("/reset-password", v1.ResetPassword)

	users := app.Group("/users")
	users.Get("/:username", v1.GetUserByUsername)
	users.Get("/:username/stats", v1.GetUserStats)
	users.Get("/:username/followers", v1.GetUserFollowers)
	users.Get("/:username/following", v1.GetUserFollowing)

	// User Badges
	users.Get("/:username/badges", v1.GetUserBadges)

	// Private routes
	user := app.Group("/user", auth.RefreshTokenMiddleware(opt))
	user.Post("/profile", auth.CheckPerm(opt, "create_comment"), v1.GetProfile)
	user.Put("/update/profile/me", auth.CheckPerm(opt, "create_comment"), v1.UpdateUserProfile)
	user.Put("/update/notification/me", auth.CheckPerm(opt, "create_comment"), v1.UpdateUserNotificationPrefrences)
	user.Put("/update/customization/me", auth.CheckPerm(opt, "create_comment"), v1.UpdateUserCustomization)
	user.Put("/update/account/me", auth.CheckPerm(opt, "create_comment"), v1.UpdateUserAccount)
	user.Delete("/account/delete/me", auth.CheckPerm(opt, "create_comment"), v1.DeleteUserAccount)

	// follow
	user.Post("/:username/follow", auth.CheckPerm(opt, "create_comment"), v1.FollowUser)
	user.Post("/:username/unfollow", auth.CheckPerm(opt, "create_comment"), v1.UnfollowUser)

	// user notifications
	user.Get("/notifications/me", auth.CheckPerm(opt, "create_comment"), v1.GetUserNotifications)
	user.Post("/notification/me/:notificationId", auth.CheckPerm(opt, "create_comment"), v1.GetUserNotificationID)

	go func() {
		<-ctx.Done()
		rclient.Close(log)
		log.Close()
	}()
}
