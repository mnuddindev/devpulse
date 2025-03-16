package v1

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/internal/auth"
	"github.com/mnuddindev/devpulse/internal/models"
	"github.com/mnuddindev/devpulse/pkg/logger"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"gorm.io/gorm"
)

var (
	DB       *gorm.DB
	Redis    *storage.RedisClient
	Logger   *logger.Logger
	EmailCfg = utils.EmailConfig{
		SMTPHost:     "0.0.0.0",
		SMTPPort:     1025,
		SMTPUsername: "",
		SMTPPassword: "",
		AppURL:       "http://localhost:3000",
		FromEmail:    "no-reply@devpulse.com",
	}
	Validator = utils.NewValidator()
)

func Refresh(c *fiber.Ctx) error {
	ipKey := "refresh:ip:" + c.IP()
	count, err := Redis.Get(c.Context(), ipKey).Int()
	if err == nil && count >= 5 {
		Logger.Warn(c.Context()).WithFields("ip", c.IP()).Logs("Refresh rate limit exceeded")
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Too many refresh attempts. Try again later.",
		})
	}
	Redis.Incr(c.Context(), ipKey)
	Redis.Expire(c.Context(), ipKey, 15*time.Minute)

	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		Logger.Warn(c.Context()).Logs("No refresh token provided")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Refresh token required",
		})
	}

	refreshKey := "refresh:" + refreshToken
	refreshDataJSON, err := Redis.Get(c.Context(), refreshKey).Result()
	if err != nil || refreshDataJSON == "" {
		Logger.Warn(c.Context()).WithFields("key", refreshKey).Logs("Invalid or expired refresh token")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired refresh token",
		})
	}

	var refreshData map[string]interface{}
	if err := json.Unmarshal([]byte(refreshDataJSON), &refreshData); err != nil {
		Logger.Error(c.Context()).WithFields("error", err).Logs("Failed to parse refresh token data")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process refresh",
		})
	}

	userID, ok := refreshData["user_id"].(string)
	if !ok || userID == "" {
		Logger.Warn(c.Context()).WithFields("key", refreshKey).Logs("Invalid refresh token data")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid refresh token",
		})
	}

	if ip, ok := refreshData["ip"].(string); !ok || ip != c.IP() {
		Logger.Warn(c.Context()).WithFields("user_id", userID).Logs("Refresh token used from different IP")
		Redis.Del(c.Context(), refreshKey) // Revoke on IP mismatch
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid refresh token (IP mismatch)",
		})
	}

	user, err := models.GetUserBy(c.Context(), Redis, DB, "id = ?", []interface{}{userID}, "")
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).Logs("Failed to fetch user for refresh")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process refresh",
		})
	}

	accessToken, err := auth.GenerateAccessToken(user.ID.String(), user.RoleID.String())
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).Logs("Failed to generate new access token")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process refresh",
		})
	}
	newRefreshToken := auth.GenerateRefreshToken()

	newRefreshKey := "refresh:" + newRefreshToken
	newRefreshData := map[string]interface{}{
		"user_id": user.ID.String(),
		"ip":      c.IP(),
	}
	newRefreshJSON, _ := json.Marshal(newRefreshData)
	if err := Redis.Set(c.Context(), newRefreshKey, newRefreshJSON, 7*24*time.Hour).Err(); err != nil {
		Logger.Warn(c.Context()).WithFields("key", newRefreshKey).Logs(fmt.Sprintf("Failed to store new refresh token: %v", err))
	}
	Redis.Del(c.Context(), refreshKey)

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	Logger.Info(c.Context()).WithFields("user_id", user.ID).Logs("Access token refreshed successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Token refreshed successfully",
	})
}
