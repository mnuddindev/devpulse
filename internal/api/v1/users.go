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

// GetUserByUsername returns a user by username
func GetUserByUsername(c *fiber.Ctx) error {
	username := c.Params("username")
	if username == "" {
		Logger.Warn(c.Context()).Logs("Missing username query parameter in GetPublicUserProfile")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Username is required",
			"status": fiber.StatusBadRequest,
		})
	}

	if len(username) < 3 || len(username) > 255 {
		Logger.Warn(c.Context()).WithFields("username", username).Logs("Invalid username length in GetPublicUserProfile")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Username must be between 3 and 255 characters",
			"status": fiber.StatusBadRequest,
		})
	}

	cacheKey := "public_user:" + username
	cachedProfile, err := Redis.Get(c.Context(), cacheKey).Result()
	if err == nil {
		var publicUser models.User
		if err := json.Unmarshal([]byte(cachedProfile), &publicUser); err == nil {
			Logger.Info(c.Context()).WithFields("username", username).Logs("Public user profile served from cache")
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"message": "Public profile retrieved successfully",
				"status":  fiber.StatusOK,
				"user":    publicUser,
			})
		}
		Logger.Warn(c.Context()).WithFields("error", err, "username", username).Logs("Failed to unmarshal cached public user")
	}

	user, err := models.GetUserBy(c.Context(), Redis, DB, "username = ?", []interface{}{username}, "")
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).WithFields("username", username).Logs("Failed to fetch user by username")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "User not found",
			"status": fiber.StatusNotFound,
		})
	}

	userJSON, _ := json.Marshal(user)
	if err := Redis.Set(c.Context(), cacheKey, userJSON, 5*time.Minute).Err(); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("username", username).Logs("Failed to cache public user profile")
	}

	profileResponse := fiber.Map{
		"id":                       user.ID,
		"username":                 user.Username,
		"email":                    user.Email,
		"name":                     user.Profile.Name,
		"bio":                      user.Profile.Bio,
		"avatar_url":               user.Profile.AvatarURL,
		"job_title":                user.Profile.JobTitle,
		"employer":                 user.Profile.Employer,
		"location":                 user.Profile.Location,
		"social_links":             user.Profile.SocialLinks,
		"current_learning":         user.Profile.CurrentLearning,
		"available_for":            user.Profile.AvailableFor,
		"currently_hacking_on":     user.Profile.CurrentlyHackingOn,
		"pronouns":                 user.Profile.Pronouns,
		"education":                user.Profile.Education,
		"brand_color":              user.Settings.BrandColor,
		"posts_count":              user.Stats.PostsCount,
		"comments_count":           user.Stats.CommentsCount,
		"likes_count":              user.Stats.LikesCount,
		"bookmarks_count":          user.Stats.BookmarksCount,
		"last_seen":                user.Stats.LastSeen,
		"theme_preference":         user.Settings.ThemePreference,
		"base_font":                user.Settings.BaseFont,
		"site_navbar":              user.Settings.SiteNavbar,
		"content_editor":           user.Settings.ContentEditor,
		"content_mode":             user.Settings.ContentMode,
		"created_at":               user.CreatedAt,
		"updated_at":               user.UpdatedAt,
		"skills":                   user.Profile.Skills,
		"interests":                user.Profile.Interests,
		"badges":                   user.Badges,
		"roles":                    user.Role,
		"followers":                user.Followers,
		"following":                user.Following,
		"notifications":            user.Notifications,
		"notification_preferences": user.NotificationPreferences,
		"previous_passwords":       user.PreviousPasswords,
		"last_password_change":     user.LastPasswordChange,
	}

	Logger.Info(c.Context()).WithFields("username", username).Logs("Public user profile retrieved successfully")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Public profile retrieved successfully",
		"status":  fiber.StatusOK,
		"user":    profileResponse,
	})
}

// GetUserStats returns user statistics
func GetUserStats(c *fiber.Ctx) error {
	return nil
}

// FollowUser follows a user
func FollowUser(c *fiber.Ctx) error {
	type FollowRequest struct {
		UserID string `json:"user_id" validate:"required,uuid"`
	}

	userIDRaw, ok := c.Locals("user_id").(string)
	if !ok || userIDRaw == "" {
		Logger.Warn(c.Context()).Logs("FollowUser attempted without user_id in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	followerID, err := uuid.Parse(userIDRaw)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "user_id", userIDRaw).Logs("Invalid user_id format in FollowUser")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	var req FollowRequest
	if err := utils.StrictBodyParser(c, &req); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).Logs("Invalid request body in FollowUser")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	if err := Validator.Validate(req); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).Logs("Validation failed in FollowUser")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	followingID, err := uuid.Parse(req.UserID)
	if err != nil {
		Logger.Warn(c.Context()).WithFields("error", err, "following_id", req.UserID).Logs("Invalid target user_id format")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid target user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	if followerID == followingID {
		Logger.Warn(c.Context()).WithFields("user_id", followerID).Logs("User attempted to follow themselves")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Cannot follow yourself",
			"status": fiber.StatusBadRequest,
		})
	}

	allowed := RateLimitting(c, followerID.String(), 5*time.Minute, 10, "follow_user_rate")
	if !allowed {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  "Too many update attempts, try again later",
			"status": fiber.StatusTooManyRequests,
		})
	}

	userKey := "user:" + followerID.String()
	var followU *models.User
	followerUser, err := Redis.Get(c.Context(), userKey).Result()
	if err == nil {
		followU = &models.User{}
		if err := json.Unmarshal([]byte(followerUser), followU); err != nil {
			Logger.Warn(c.Context()).WithFields("error", err, "userID", userIDRaw).Logs("Failed to unmarshal cached user from Redis")
			followU = nil
		}
	} else {
		followU, err = models.GetUserBy(c.Context(), Redis, DB, "id = ?", []interface{}{followerID})
		if err != nil {
			Logger.Warn(c.Context()).WithFields("error", err, "user_id", followerID).Logs("Follower not found")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":  "User not found",
				"status": fiber.StatusUnauthorized,
			})
		}
	}

	following, err := models.GetUserBy(c.Context(), Redis, DB, "id = ?", []interface{}{followingID})
	if err != nil {
		Logger.Warn(c.Context()).WithFields("error", err, "following_id", followingID).Logs("Target user not found")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "Target user not found",
			"status": fiber.StatusNotFound,
		})
	}

	if err := DB.WithContext(c.Context()).Model(&followU).Association("Following").Find(&followU.Following); err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "user_id", followerID).Logs("Failed to load Following relationship")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to process request",
			"status": fiber.StatusInternalServerError,
		})
	}

	for _, f := range followU.Following {
		if f.ID == followingID {
			Logger.Info(c.Context()).WithFields("user_id", followerID, "following_id", followingID).Logs("User already following target")
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"message": "Already following this user",
				"status":  fiber.StatusOK,
			})
		}
	}

	if err := DB.WithContext(c.Context()).Model(&followU).Association("Following").Append(following); err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "user_id", followerID, "following_id", followingID).Logs("Failed to add following relationship")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to follow user",
			"status": fiber.StatusInternalServerError,
		})
	}

	followerKey := "user:" + followerID.String()
	followingKey := "user:" + followingID.String()
	Redis.Del(c.Context(), followerKey, followingKey)

	Logger.Info(c.Context()).WithFields("user_id", followerID, "following_id", followingID).Logs("User followed successfully")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User followed successfully",
		"status":  fiber.StatusOK,
	})
}
