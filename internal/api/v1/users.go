package v1

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/internal/models"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/redis/go-redis/v9"
)

// ForgotPassword handles the password reset request
func ForgotPassword(c *fiber.Ctx) error {
	type ForgotPasswordRequest struct {
		Email string `json:"email" validate:"required,email,max=100"`
	}
	var data ForgotPasswordRequest
	if err := utils.StrictBodyParser(c, &data); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_email", data.Email).Logs("Invalid request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	if err := Validator.Validate(data); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_email", data.Email).Logs("Validation failed")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	user, err := models.GetUserBy(c.Context(), Redis, DB, "email = ?", []interface{}{data.Email}, "")
	if err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("email", data.Email).Logs("User not found in ForgotPassword")
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "If the email exists, a reset link has been sent",
			"status":  fiber.StatusOK,
		})
	}

	allowed := RateLimitting(c, user.ID.String(), 1*time.Minute, 5, "forgot_password_rate:")
	if !allowed {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  "Too many update attempts, try again later",
			"status": fiber.StatusTooManyRequests,
		})
	}

	token, _ := utils.GenerateRandomToken(26, 40)
	gotp, err := utils.GenerateOTP()
	if err != nil {
		Logger.Error(c.Context()).Logs(fmt.Sprintf("Failed to generate OTP: %v", err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate activation code",
		})
	}

	tokenKey := "reset_token:" + token
	expiresIn := 1 * time.Hour

	if err := Redis.Set(c.Context(), tokenKey, user.ID.String(), expiresIn).Err(); err != nil {
		Logger.Error(c.Context()).WithFields("error", err).WithFields("user_id", user.ID).Logs("Failed to store reset token in Redis")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to process request",
			"status": fiber.StatusInternalServerError,
		})
	}

	userKey := "user:" + user.ID.String()
	Redis.Del(c.Context(), userKey)
	key := "user:" + token
	if err := Redis.Set(c.Context(), key, user.ID, 1*time.Hour).Err(); err != nil {
		Logger.Warn(c.Context()).Logs(fmt.Sprintf("Failed to cache user in Redis: %v, key: %s", err, key))
	} else {
		Logger.Info(c.Context()).Logs(fmt.Sprintf("User cached in Redis: %s", key))
	}

	go func() {
		if err := utils.SendActivationEmail(c.Context(), EmailCfg, user.Email, user.Username, token, gotp, Logger); err != nil {
			Logger.Warn(c.Context()).Logs(fmt.Sprintf("Email sending failed but user created: %v", err))
		} else {
			Logger.Info(c.Context()).Logs(fmt.Sprintf("Activation email sent successfully for user: %s", user.Username))
		}
	}()

	Logger.Info(c.Context()).WithFields("user_id", user.ID).Logs("Password reset token generated and stored in Redis")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "If the email exists, a reset link has been sent",
		"status":  fiber.StatusOK,
	})
}

// ResetPassword handles the password reset process
func ResetPassword(c *fiber.Ctx) error {
	type ResetPasswordRequest struct {
		Token           string `json:"token" validate:"required,min=26,max=40"`
		NewPassword     string `json:"new_password" validate:"required,min=6"`
		ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
	}

	var req ResetPasswordRequest
	if err := utils.StrictBodyParser(c, &req); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).Logs("Invalid request body in ResetPassword")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	if err := Validator.Validate(req); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).Logs("Validation failed in ResetPassword")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	if utils.ContainsInvalidChars(req.NewPassword) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "password contains invalid characters",
			"status":  fiber.StatusBadRequest,
		})
	}

	tokenKey := "reset_token:" + req.Token
	token, err := Redis.Get(c.Context(), tokenKey).Result()
	if err == redis.Nil {
		Logger.Warn(c.Context()).WithFields("token", req.Token).Logs("Invalid or expired reset token")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid or expired reset token",
			"status": fiber.StatusBadRequest,
		})
	} else if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).WithFields("token", req.Token).Logs("Failed to fetch reset token from Redis")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to process request",
			"status": fiber.StatusInternalServerError,
		})
	}

	userIDRaw, err := Redis.Get(c.Context(), "user:"+token).Result()
	if err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).Logs("User not found or expired")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired User data",
		})
	}

	userID, err := uuid.Parse(userIDRaw)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "user_id_raw", userIDRaw).Logs("Invalid user ID in reset token")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to process request",
			"status": fiber.StatusInternalServerError,
		})
	}

	allowed := RateLimitting(c, userIDRaw, 1*time.Minute, 5, "forgot_password_rate:")
	if !allowed {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  "Too many update attempts, try again later",
			"status": fiber.StatusTooManyRequests,
		})
	}

	user, err := models.GetUserBy(c.Context(), Redis, DB, "id = ?", []interface{}{userID}, "")
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("User not found in ResetPassword")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid or expired reset token", // Don’t leak user existence
			"status": fiber.StatusBadRequest,
		})
	}

	if !utils.IsStrongPassword(req.NewPassword) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "new password does not meet strength requirements",
			"status":  fiber.StatusBadRequest,
		})
	}

	if utils.IsPasswordReused(user.PreviousPasswords, req.NewPassword) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "new password cannot match previous passwords",
			"status":  fiber.StatusBadRequest,
		})
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Failed to hash new password")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to process password",
			"status": fiber.StatusInternalServerError,
		})
	}

	updatedUser, err := models.UpdateUser(
		c.Context(),
		Redis,
		DB,
		userID,
		models.WithPassword(string(hashedPassword)),
		models.WithPreviousPasswords(utils.UpdatePreviousPasswords(user.PreviousPasswords, user.Password)),
		models.WithPasswordChangedAt(time.Now()),
	)

	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Failed to update user password")
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update password"})
	}

	if err := Redis.Del(c.Context(), tokenKey).Err(); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("token", req.Token).Logs("Failed to delete reset token from Redis")
	}

	if err := Redis.Del(c.Context(), tokenKey).Err(); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("token", req.Token).Logs("Failed to delete reset token from Redis")
	}

	userKey := "user:" + userID.String()
	Redis.Del(c.Context(), userKey)
	userJSON, _ := json.Marshal(updatedUser)
	if err := Redis.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Failed to update Redis cache")
	}

	Logger.Info(c.Context()).WithFields("user_id", userID).Logs("Password reset successfully")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Password reset successfully. Please try logging in",
		"status":  fiber.StatusOK,
	})
}
