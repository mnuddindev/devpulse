package auth

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/internal/models"
	"github.com/redis/go-redis/v9"
)

func RefreshTokenMiddleware(opt Options) fiber.Handler {
	return func(c *fiber.Ctx) error {
		publicRoutes := map[string]bool{
			"/register":      true,
			"/activate":      true,
			"/login":         true,
			"/logout":        true,
			"/refresh-token": true,
		}
		path := c.Path()
		if publicRoutes[path] {
			return c.Next()
		}

		accessToken := c.Cookies("access_token")
		refreshToken := c.Cookies("refresh_token")
		opt.Logger.Info(c.Context()).WithFields("access_token", accessToken).WithFields("refresh_token", refreshToken).Logs("Tokens received in middleware")

		if accessToken != "" {
			accessTokenKey := "blacklist:access:" + accessToken
			if opt.Rclient.Exists(c.Context(), accessTokenKey).Val() > 0 {
				opt.Logger.Warn(c.Context()).WithFields("token", accessToken).Logs("Attempted use of blacklisted access token")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Access token has been invalidated",
				})
			}
		}
		if refreshToken != "" {
			refreshTokenKey := "blacklist:refresh:" + refreshToken
			if opt.Rclient.Exists(c.Context(), refreshTokenKey).Val() > 0 {
				opt.Logger.Warn(c.Context()).WithFields("token", refreshToken).Logs("Attempted use of blacklisted refresh token")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Refresh token has been invalidated",
				})
			}
		}

		if accessToken == "" {
			opt.Logger.Debug(c.Context()).Logs("No access token found, attempting refresh")
			newAccessToken, err := handleTokenRefresh(c, opt, refreshToken)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token refresh failed"})
			}
			accessToken = newAccessToken
		}

		claims, err := VerifyToken(accessToken)
		if err != nil {
			if err.Error() == "token has expired" {
				opt.Logger.Debug(c.Context()).Logs("Access token expired, attempting refresh")
				newAccessToken, err := handleTokenRefresh(c, opt, refreshToken)
				if err != nil {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token refresh failed"})
				}
				accessToken = newAccessToken
				claims, err = VerifyToken(accessToken)
				if err != nil {
					opt.Logger.Warn(c.Context()).WithFields("error", err).Logs("Invalid access token after refresh")
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid access token"})
				}
			}
			opt.Logger.Warn(c.Context()).WithFields("error", err).Logs("Access token invalid")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid access token",
			})
		}

		var user *models.User
		user, err = models.GetUserBy(c.Context(), opt.Rclient, opt.DB, "id = ?", []interface{}{claims.UserID}, "")
		if err != nil {
			opt.Logger.Warn(c.Context()).WithFields("user_id", claims.UserID).Logs("User not found")
			c.ClearCookie("access_token")
			c.ClearCookie("refresh_token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not found",
			})
		}

		userKey := "user:" + user.ID.String()
		cachedUser, err := opt.Rclient.Get(c.Context(), userKey).Result()
		if err == nil && cachedUser != "" {
			user = &models.User{}
			if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
				opt.Logger.Warn(c.Context()).WithFields("error", err).Logs("Failed to unmarshal cached user")
				user = nil
			}
		}
		if err == redis.Nil || user == nil {
			userJSON, _ := json.Marshal(user)
			opt.Rclient.Set(c.Context(), userKey, userJSON, 30*time.Minute)
		} else if err != nil {
			opt.Logger.Warn(c.Context()).WithFields("user_id", claims.UserID).Logs("Redis error fetching user")
			user, err = models.GetUserBy(c.Context(), opt.Rclient, opt.DB, "id = ?", []interface{}{uuid.MustParse(claims.UserID)}, "")
			if err != nil {
				opt.Logger.Warn(c.Context()).WithFields("user_id", claims.UserID).Logs("User not found during access token validation")
				c.ClearCookie("access_token")
				c.ClearCookie("refresh_token")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "User not found",
				})
			}
		}

		c.Locals("user_id", claims.UserID)

		if claims.RoleID != user.RoleID.String() {
			opt.Logger.Warn(c.Context()).WithFields("user_id", user.ID).WithFields("token_role", claims.RoleID).WithFields("user_role", user.RoleID).Logs("Role mismatch")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Role mismatch",
			})
		}

		opt.Logger.Info(c.Context()).WithFields("user_id", claims.UserID).Logs(fmt.Sprintf("User authenticated for route: %s", path))
		return c.Next()
	}
}

// refreshTokens generates new tokens
func handleTokenRefresh(c *fiber.Ctx, cfg Options, refreshToken string) (string, error) {
	if refreshToken == "" {
		cfg.Logger.Warn(c.Context()).Logs("Refresh token missing")
		return "", c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Refresh token missing"})
	}

	refreshTokenKey := "blacklist:refresh:" + refreshToken
	if cfg.Rclient.Exists(c.Context(), refreshTokenKey).Val() > 0 {
		cfg.Logger.Warn(c.Context()).WithFields("token", refreshToken).Logs("Attempted use of blacklisted refresh token")
		return "", c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Refresh token has been invalidated",
			"status": fiber.StatusUnauthorized,
		})
	}

	refreshKey := "refresh:" + refreshToken
	refreshDataJSON, err := cfg.Rclient.Get(c.Context(), refreshKey).Result()
	if err != nil || refreshDataJSON == "" {
		cfg.Logger.Warn(c.Context()).WithFields("token", refreshToken).Logs("Invalid/expired refresh token")
		return "", c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid/expired refresh token"})
	}

	var refreshData map[string]interface{}
	if err := json.Unmarshal([]byte(refreshDataJSON), &refreshData); err != nil {
		cfg.Logger.Error(c.Context()).WithFields("error", err).Logs("Failed to parse refresh data")
		return "", c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process refresh"})
	}

	userID, ok := refreshData["user_id"].(string)
	if !ok || userID == "" {
		cfg.Logger.Warn(c.Context()).Logs("Invalid refresh token data")
		return "", c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid refresh token"})
	}

	if ip, ok := refreshData["ip"].(string); !ok || ip != c.IP() {
		cfg.Logger.Warn(c.Context()).WithFields("user_id", userID).Logs("IP mismatch")
		cfg.Rclient.Del(c.Context(), refreshKey)
		return "", c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "IP mismatch"})
	}

	var user *models.User
	cachedUser, err := cfg.Rclient.Get(c.Context(), "user:"+userID).Result()
	if err == nil {
		user := &models.User{}
		if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
			cfg.Logger.Warn(c.Context()).WithFields("error", err, "user_id", userID).Logs("Failed to unmarshal cached user from Redis")
			user = nil
		}
	}

	if err == redis.Nil || user == nil {
		user, err = models.GetUserBy(c.Context(), cfg.Rclient, cfg.DB, "id = ?", []interface{}{uuid.MustParse(userID)}, "Role")
		if err != nil {
			cfg.Logger.Warn(c.Context()).WithFields("user_id", userID).Logs("User not found")
			c.ClearCookie("access_token")
			c.ClearCookie("refresh_token")
			return "", c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not found"})
		}

		userJson, err := json.Marshal(user)
		if err != nil {
			cfg.Logger.Warn(c.Context()).WithFields("error", err, "user_id", userID).Logs("Failed to marshal cached user from Redis")
		} else {
			if err := cfg.Rclient.Set(c.Context(), "user:"+userID, userJson, 30*time.Minute).Err(); err != nil {
				cfg.Logger.Warn(c.Context()).WithFields("error", err, "user_id", refreshData["user_id"]).Logs("refreshData")
			}
		}
	} else if err != nil {
		cfg.Logger.Warn(c.Context()).WithFields("error", err).Logs("Redis error fetching user")
		user, err = models.GetUserBy(c.Context(), cfg.Rclient, cfg.DB, "id = ?", []interface{}{refreshData["user_id"]}, "")
		if err != nil {
			cfg.Logger.Warn(c.Context()).WithFields("user_id", userID).Logs("User not found")
			c.ClearCookie("access_token")
			c.ClearCookie("refresh_token")
			return "", c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not found"})
		}
	}

	if roleID, ok := refreshData["role_id"].(string); ok && roleID != "" && uuid.MustParse(roleID) != user.RoleID {
		cfg.Logger.Warn(c.Context()).WithFields("user_id", user.ID).WithFields("token_role", roleID).WithFields("user_role", user.RoleID).Logs("Role mismatch")
		return "", c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Role mismatch"})
	}

	newAccessToken, err := GenerateAccessToken(user.ID.String(), user.RoleID.String())
	if err != nil {
		cfg.Logger.Error(c.Context()).WithFields("error", err).Logs("Failed to generate access token")
		return "", c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to refresh"})
	}
	newRefreshToken := GenerateRefreshToken()

	newRefreshKey := "refresh:" + newRefreshToken
	newRefreshData := map[string]interface{}{
		"user_id": user.ID,
		"ip":      c.IP(),
	}
	newRefreshJSON, _ := json.Marshal(newRefreshData)
	cfg.Rclient.Set(c.Context(), newRefreshKey, newRefreshJSON, 7*24*time.Hour)
	cfg.Rclient.Del(c.Context(), refreshKey)

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    newAccessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
		// SameSite: "strict",
		// Path:     "/",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		// SameSite: "strict",
		// Path:     "/",
	})

	c.Locals("user_id", user.ID.String())

	cfg.Logger.Info(c.Context()).WithFields("user_id", userID).Logs("Tokens refreshed")
	return newAccessToken, nil
}

// GetRolePermissions extracts permissions from a role
func GetRolePermissions(role *models.Role) []string {
	var perms []string
	for _, perm := range role.Permissions {
		perms = append(perms, perm.Name)
	}
	return perms
}
