package v1

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/internal/auth"
	"github.com/mnuddindev/devpulse/internal/models"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RateLimitting(c *fiber.Ctx, userID string, rateTTL time.Duration, maxUpdates int, prefix string) bool {
	rateKey := prefix + userID
	count, err := Redis.Get(c.Context(), rateKey).Int()
	if err == redis.Nil {
		count = 0
	} else if err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Failed to check rate limit")
	}
	if count >= maxUpdates {
		Logger.Warn(c.Context()).WithFields("user_id", userID).Logs("Rate limit exceeded")
		return false
	}

	pipe := Redis.TxPipeline()
	pipe.Incr(c.Context(), rateKey)
	pipe.Expire(c.Context(), rateKey, rateTTL)
	if _, err := pipe.Exec(c.Context()); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Failed to update rate limit")

	}
	return true
}

func Register(c *fiber.Ctx) error {
	if utils.IsLoggedIn(c) {
		Logger.Warn(c.Context()).Logs("User already logged in, registration not allowed")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "Already logged in",
			"message": "You are already authenticated. Please log out first if you want to register a new account.",
		})
	}
	type UserInput struct {
		AvatarURL       string `json:"avatar_url" validate:"omitempty,url"`
		Name            string `json:"name" validate:"required,min=8,max=100"`
		Username        string `json:"username" validate:"required,min=3,max=50,alphanum"`
		Email           string `json:"email" validate:"required,email,max=100"`
		Password        string `json:"password" validate:"required,min=6,eqfield=ConfirmPassword"`
		ConfirmPassword string `json:"confirm_password" validate:"required,min=6"`
	}
	ui := new(UserInput)
	if err := utils.StrictBodyParser(c, &ui); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).Logs(fmt.Sprintf("Failed to parse request body: %v", err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	if err := Validator.Validate(ui); err != nil {
		Logger.Warn(c.Context()).WithFields("errors", err).Logs(fmt.Sprintf("Validation failed: %s", err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"details": err,
		})
	}

	if utils.ContainsInvalidChars(ui.Password) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "password contains invalid characters",
			"status":  fiber.StatusBadRequest,
		})
	}

	ui.Email = strings.ToLower(strings.TrimSpace(ui.Email))

	hashedPass, err := utils.HashPassword(ui.Password)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).Logs(fmt.Sprintf("Failed to hash password: %v", err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

	gotp, err := utils.GenerateOTP(10)
	if err != nil {
		Logger.Error(c.Context()).Logs(fmt.Sprintf("Failed to generate OTP: %v", err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate activation code",
		})
	}

	if !utils.IsStrongPassword(ui.Password) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "new password does not meet strength requirements",
			"status":  fiber.StatusBadRequest,
		})
	}

	user, err := models.NewUser(c.Context(), Redis, DB, ui.Username, ui.Email, hashedPass, gotp, models.WithName(ui.Name), models.WithAvatarURL(ui.AvatarURL))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			Logger.Warn(c.Context()).Logs(fmt.Sprintf("Duplicate username or email: %s", ui.Email))
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Username or email already exists",
			})
		}
		Logger.Error(c.Context()).Logs(fmt.Sprintf("Failed to create user: %v", err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	token, _ := utils.GenerateRandomToken(64, 124)
	otp, err := utils.HashPassword(gotp)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).Logs("Failed to hash OTP")
	} else {
		otpKey := fmt.Sprintf("otp:%s", token)
		if err := Redis.Set(c.Context(), otpKey, otp, 24*time.Hour).Err(); err != nil {
			Logger.Warn(c.Context()).WithFields("key", otpKey).Logs(fmt.Sprintf("Failed to store OTP in Redis: %v", err))
		} else {
			Logger.Info(c.Context()).WithFields("key", otpKey).Logs("OTP stored in Redis")
		}
	}

	if err := utils.SendActivationEmail(c.Context(), EmailCfg, ui.Email, ui.Username, token, gotp, Logger); err != nil {
		Logger.Warn(c.Context()).Logs(fmt.Sprintf("Email sending failed but user created: %v", err))
	} else {
		Logger.Info(c.Context()).Logs(fmt.Sprintf("Activation email sent successfully for user: %s", ui.Username))
	}

	key := "user:" + token
	userJSON, err := json.Marshal(user)
	if err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).Logs("Failed to serialize user data")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to serialize user data",
		})
	}
	if err := Redis.Set(c.Context(), key, userJSON, 10*time.Minute).Err(); err != nil {
		Logger.Warn(c.Context()).Logs(fmt.Sprintf("Failed to cache user in Redis: %v, key: %s", err, key))
	} else {
		Logger.Info(c.Context()).Logs(fmt.Sprintf("User cached in Redis: %s", key))
	}

	// Log success
	Logger.Info(c.Context()).Logs(fmt.Sprintf("User registered successfully: %s (ID: %s)", ui.Username, user.ID.String()))

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Registration successful. Check your email to activate your account.",
		"user": fiber.Map{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

// ActivateUser verifies OTP and activates the user
func ActivateUser(c *fiber.Ctx) error {
	token := c.Query("token")

	type ActivateRequest struct {
		OTP string `json:"otp" validate:"required"`
	}

	var ar ActivateRequest
	if err := utils.StrictBodyParser(c, &ar); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).Logs(fmt.Sprintf("Failed to parse activation request: %v", err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	if err := Validator.Validate(ar); err != nil {
		Logger.Warn(c.Context()).WithFields("errors", err).Logs(fmt.Sprintf("Validation failed: %s", err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"details": err,
		})
	}

	cachedUser, err := Redis.Get(c.Context(), "user:"+token).Result()
	if err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).Logs("User not found or expired")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired User data",
		})
	}

	var marshedUser models.User
	err = json.Unmarshal([]byte(cachedUser), &marshedUser)
	if err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).Logs("Failed to deserialize user data")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to deserialize user data",
		})
	}

	user, err := models.GetUserBy(c.Context(), Redis, DB, "email = ?", []interface{}{marshedUser.Email})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			Logger.Warn(c.Context()).WithFields("email", marshedUser.Email).Logs("User not found")
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		Logger.Error(c.Context()).WithFields("error", err).Logs("Failed to fetch user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process activation",
		})
	}

	otpKey := "otp:" + token
	otpHash, err := Redis.Get(c.Context(), otpKey).Result()
	if err != nil || otpHash == "" {
		Logger.Warn(c.Context()).WithFields("key", otpKey).Logs("OTP not found or expired")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired activation code",
		})
	}

	fmt.Println("OTP Hash:", otpHash)
	fmt.Print("OTP:", ar.OTP)
	if err := utils.ComparePasswords(otpHash, ar.OTP); err != nil {
		Logger.Warn(c.Context()).WithFields("email", user.Email).Logs("Invalid OTP provided")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid activation code",
		})
	}

	updatedUser, err := models.UpdateUser(c.Context(), Redis, DB, user.ID, models.WithIsActive(true))
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).Logs("Failed to activate user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to activate account",
		})
	}

	notificationTitle := []string{
		"Welcome to DEVPULSE! 👋 I'm Heart, the community mascot and I'm here to help get you started. Let's begin by setting up your profile!",
		"Heart here again! 👋 DEVPULSE is a friendly community. Why not introduce yourself by leaving a comment in the welcome thread!",
		"You're on a roll! 🎉 Do you have a Facebook account? Consider connecting it.",
		"Hi, it's me again! 👋 Now that you're a part of the DEVPULSE community, let's focus on personalizing your content. You can start by following some tags to help customize your feed! 🎉",
		"Heart here! 👋 Did you know that that you can customize your DEVPULSE experience? Try changing your font and theme and find the best style for you!",
		"Heart here! 👋 I noticed that you haven't asked a question or started a discussion yet. It's easy to do both of these; just click on 'Write a Post' in the sidebar of the tag page to get started!",
	}

	for _, title := range notificationTitle {
		_, err := models.NewNotification(c.Context(), Redis, DB, updatedUser.ID, "all", title)
		if err != nil {
			Logger.Warn(c.Context()).WithFields("error", err, "user_id", updatedUser.ID).Logs("Failed to create welcome notification")
		}
	}

	Redis.Del(c.Context(), otpKey)
	Redis.Del(c.Context(), "user:"+token)
	Logger.Info(c.Context()).WithFields("user_id", user.ID).Logs(fmt.Sprintf("User activated successfully: %s", user.Username))

	key := "user:" + user.Email
	userJSON, err := json.Marshal(updatedUser)
	if err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).Logs("Failed to serialize user data")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to serialize user data",
		})
	}
	if err := Redis.Set(c.Context(), key, userJSON, 0).Err(); err != nil {
		Logger.Warn(c.Context()).Logs(fmt.Sprintf("Failed to cache user in Redis: %v, key: %s", err, key))
	} else {
		Logger.Info(c.Context()).Logs(fmt.Sprintf("User cached in Redis: %s", key))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Account activated successfully",
		"user": fiber.Map{
			"id":       updatedUser.ID,
			"username": updatedUser.Username,
			"email":    updatedUser.Email,
			"message":  "Your account has been activated. Please log in now!",
		},
	})
}

// Login ensures user can login to his account.
func Login(c *fiber.Ctx) error {
	if utils.IsLoggedIn(c) {
		Logger.Warn(c.Context()).Logs("User already logged in, login not allowed")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "Already logged in",
			"message": "You are already authenticated. Please log out first if you want to log in again.",
		})
	}
	type LoginRequest struct {
		Email    string `json:"email" validate:"required,email,max=100"`
		Password string `json:"password" validate:"required,min=6,max=100"`
	}

	var lr LoginRequest
	if err := utils.StrictBodyParser(c, &lr); err != nil {
		Logger.Warn(c.Context()).Logs(fmt.Sprintf("Failed to parse login request body: %v", err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	ipKey := "login:ip:" + c.IP()
	count, err := Redis.Get(c.Context(), ipKey).Int()
	if err != nil && count >= 5 {
		Logger.Warn(c.Context()).WithFields("ip", c.IP()).Logs("Login rate limit exceeded")
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Too many login attempts. Try again later.",
		})
	}
	Redis.Incr(c.Context(), ipKey)
	Redis.Expire(c.Context(), ipKey, 15*time.Minute)

	if err := Validator.Validate(lr); err != nil {
		Logger.Warn(c.Context()).WithFields("errors", err).Logs("Login validation failed")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Validation failed",
			"details": fiber.Map{
				"errors": err,
			},
		})
	}

	lr.Email = strings.ToLower(strings.TrimSpace(lr.Email))

	user, err := models.GetUserBy(c.Context(), Redis, DB, "email = ?", []interface{}{lr.Email})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			Logger.Warn(c.Context()).WithFields("email", user.Email).Logs("User not found")
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		Logger.Error(c.Context()).WithFields("error", err).Logs("Failed to fetch user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process activation",
		})
	}

	if !user.IsActive || !user.IsEmailVerified {
		Logger.Warn(c.Context()).WithFields("user_id", user.ID).Logs("Login attempt on inactive account")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Account not activated. Check your email.",
		})
	}

	if err := utils.ComparePasswords(user.Password, lr.Password); err != nil {
		Logger.Warn(c.Context()).WithFields("email", lr.Email).Logs("Invalid password provided")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	user.UpdateLastSeen(c.Context(), Redis, DB)

	accessToken, err := auth.GenerateAccessToken(user.ID.String(), user.RoleID.String())
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).Logs("Failed to generate access token")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process login",
		})
	}
	refreshToken := auth.GenerateRefreshToken()

	refreshKey := "refresh:" + refreshToken
	refreshData := map[string]interface{}{
		"user_id": user.ID.String(),
		"ip":      c.IP(),
	}
	refreshJSON, _ := json.Marshal(refreshData)
	if err := Redis.Set(c.Context(), refreshKey, refreshJSON, 7*24*time.Hour).Err(); err != nil {
		Logger.Warn(c.Context()).WithFields("key", refreshKey).Logs(fmt.Sprintf("Failed to store refresh token: %v", err))
	}

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
		// Secure:   true,
		// SameSite: "Strict",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		// Secure:   true,
		// SameSite: "Strict",
	})

	Redis.Del(c.Context(), ipKey)
	Redis.Del(c.Context(), "user:"+user.ID.String())

	Logger.Info(c.Context()).WithFields("user_id", user.ID).Logs(fmt.Sprintf("User logged in successfully: %s", user.Username))

	key := "user:" + user.ID.String()
	userJSON, err := json.Marshal(user)
	if err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).Logs("Failed to serialize user data")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to serialize user data",
		})
	}

	if err := Redis.Set(c.Context(), key, userJSON, 30*time.Minute).Err(); err != nil {
		Logger.Warn(c.Context()).Logs(fmt.Sprintf("Failed to cache user in Redis: %v, key: %s", err, key))
	} else {
		Logger.Info(c.Context()).Logs(fmt.Sprintf("User cached in Redis: %s", key))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login successful",
		"user": fiber.Map{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"name":     user.Profile.Name,
			"avatar":   user.Profile.AvatarURL,
		},
	})
}

// Logout ensures user logged out from the server.
func Logout(c *fiber.Ctx) error {
	accessToken := c.Cookies("access_token")
	refreshToken := c.Cookies("refresh_token")
	accessTokenKey := "blacklist:access:" + accessToken
	refreshTokenKey := "blacklist:refresh:" + refreshToken

	if accessToken != "" {
		if err := Redis.Set(c.Context(), accessTokenKey, "invalid", 15*time.Minute).Err(); err != nil {
			Logger.Warn(c.Context()).WithFields("error", err).Logs("Failed to blacklist access token in Redis")
		}
	}

	if utils.IsLoggedIn(c) {
		Logger.Warn(c.Context()).Logs("Refresh token not found in Redis")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "User not logged in",
			"status": fiber.StatusBadRequest,
		})
	}

	var refreshData map[string]interface{}
	if refreshToken != "" {
		if err := Redis.Set(c.Context(), refreshTokenKey, "invalid", 7*24*time.Hour).Err(); err != nil {
			Logger.Warn(c.Context()).WithFields("error", err).Logs("Failed to blacklist refresh token in Redis")
		}
		refreshKey := "refresh:" + refreshToken
		refreshDataJSON, err := Redis.Get(c.Context(), refreshKey).Result()
		if err == nil && refreshDataJSON != "" {
			if err := json.Unmarshal([]byte(refreshDataJSON), &refreshData); err == nil {
				if userID, ok := refreshData["user_id"].(string); ok {
					Logger.Info(c.Context()).WithFields("user_id", userID).Logs("User logged out, refresh token revoked")
				}
			}
			Redis.Del(c.Context(), refreshKey)
		} else {
			Logger.Warn(c.Context()).WithFields("refresh_key", refreshKey).Logs("Refresh token not found in Redis")
		}
	} else {
		Logger.Warn(c.Context()).Logs("No refresh token provided for logout")
	}

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})
	c.ClearCookie("access_token")
	c.ClearCookie("refresh_token")
	c.Locals("user_id", "")

	Redis.Del(c.Context(), refreshData["user_id"].(string))

	c.Set("Authorization", "")
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	c.Set("Pragma", "no-cache")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "DENY")
	c.Set("Content-Security-Policy", "default-src 'self'")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Logout successful",
		"status":  fiber.StatusOK,
	})
}

// UserProfile retrieves and returns the authenticated user’s profile, optimized with Redis caching
func GetProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		Logger.Warn(c.Context()).Logs("UserProfile attempted without user ID in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).Logs("Invalid user ID format in UserProfile")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	userKey := "user:" + uid.String()
	var user *models.User
	cachedUser, err := Redis.Get(c.Context(), userKey).Result()
	if err == nil {
		user = &models.User{}
		if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
			Logger.Warn(c.Context()).WithFields("error", err, "userID", uid).Logs("Failed to unmarshal cached user from Redis")
			user = nil
		}
	}
	if err == redis.Nil || user == nil {
		user, err = models.GetUserBy(c.Context(), Redis, DB, "id = ?", []interface{}{uid}, "")
		if err != nil {
			Logger.Error(c.Context()).WithFields("error", err, "userID", uid).Logs("Database error while fetching user profile")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to fetch user profile",
				"status": fiber.StatusInternalServerError,
			})
		}
		userJSON, err := json.Marshal(user)
		if err != nil {
			Logger.Warn(c.Context()).WithFields("error", err, "userID", uid).Logs("Failed to marshal user for Redis caching")
		} else {
			if err := Redis.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
				Logger.Warn(c.Context()).WithFields("error", err, "userID", uid).Logs("Failed to cache user profile in Redis")
			}
		}
	} else if err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "userID", uid).Logs("Redis error fetching user profile")
		user, err = models.GetUserBy(c.Context(), Redis, DB, "id = ?", []interface{}{uid})
		if err != nil {
			Logger.Error(c.Context()).WithFields("error", err, "userID", uid).Logs("Database error while fetching user profile")
			c.ClearCookie("access_token")
			c.ClearCookie("refresh_token")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to fetch user profile",
				"status": fiber.StatusInternalServerError,
			})
		}
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

	Logger.Info(c.Context()).WithFields("userID", uid).Logs("User profile retrieved successfully")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Profile retrieved successfully",
		"status":  fiber.StatusOK,
		"user":    profileResponse,
	})
}

// UpdateUserProfile updates the authenticated user’s profile with Redis caching and single-query perfection
func UpdateUserProfile(c *fiber.Ctx) error {
	userIDRaw, ok := c.Locals("user_id").(string)
	if !ok || userIDRaw == "" {
		Logger.Warn(c.Context()).Logs("UpdateUserProfile attempted without user ID in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	userID, err := uuid.Parse(userIDRaw)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "userID", userIDRaw).Logs("Invalid user ID format in UpdateUserProfile")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	allowed := RateLimitting(c, userIDRaw, 1*time.Minute, 5, "profile_update_rate:")
	if !allowed {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  "Too many update attempts, try again later",
			"status": fiber.StatusTooManyRequests,
		})
	}

	var req models.UpdateUserRequest
	if err := utils.StrictBodyParser(c, &req); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Invalid request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	if err := Validator.Validate(req); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Validation failed")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	userKey := "user:" + userIDRaw
	var user *models.User
	cachedUser, err := Redis.Get(c.Context(), userKey).Result()
	if err == nil {
		user = &models.User{}
		if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
			Logger.Warn(c.Context()).WithFields("error", err, "userID", userIDRaw).Logs("Failed to unmarshal cached user from Redis")
			user = nil
		}
	}

	var opts []models.UserOption
	updatedFields := []string{}
	if req.Username != nil {
		if err := DB.Where("username = ? AND id != ?", *req.Username, userID).First(&models.User{}).Error; err == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Username already taken"})
		}
		opts = append(opts, models.WithUsername(*req.Username))
		updatedFields = append(updatedFields, "username")
	}
	if req.Email != nil {
		if err := DB.Where("email = ? AND id != ?", *req.Email, userID).First(&models.User{}).Error; err == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Email already taken"})
		}
		opts = append(opts, models.WithEmail(*req.Email))
		updatedFields = append(updatedFields, "email")
	}
	if req.Profile != nil {
		if req.Profile.Name != nil {
			opts = append(opts, models.WithName(*req.Profile.Name))
			updatedFields = append(updatedFields, "profile.name")
		}
		if req.Profile.Bio != nil {
			opts = append(opts, models.WithBio(*req.Profile.Bio))
			updatedFields = append(updatedFields, "profile.bio")
		}
		if req.Profile.AvatarURL != nil {
			opts = append(opts, models.WithAvatarURL(*req.Profile.AvatarURL))
			updatedFields = append(updatedFields, "profile.avatar_url")
		}
		if req.Profile.JobTitle != nil {
			opts = append(opts, models.WithJobTitle(*req.Profile.JobTitle))
			updatedFields = append(updatedFields, "profile.job_title")
		}
		if req.Profile.Employer != nil {
			opts = append(opts, models.WithEmployer(*req.Profile.Employer))
			updatedFields = append(updatedFields, "profile.employer")
		}
		if req.Profile.Location != nil {
			opts = append(opts, models.WithLocation(*req.Profile.Location))
			updatedFields = append(updatedFields, "profile.location")
		}
		if req.Profile.SocialLinks != nil {
			opts = append(opts, models.WithSocialLinks(*req.Profile.SocialLinks))
			updatedFields = append(updatedFields, "profile.social_links")
		}
		if req.Profile.CurrentLearning != nil {
			opts = append(opts, models.WithCurrentLearning(*req.Profile.CurrentLearning))
			updatedFields = append(updatedFields, "profile.current_learning")
		}
		if req.Profile.AvailableFor != nil {
			opts = append(opts, models.WithAvailableFor(*req.Profile.AvailableFor))
			updatedFields = append(updatedFields, "profile.available_for")
		}
		if req.Profile.CurrentlyHackingOn != nil {
			opts = append(opts, models.WithCurrentlyHackingOn(*req.Profile.CurrentlyHackingOn))
			updatedFields = append(updatedFields, "profile.currently_hacking_on")
		}
		if req.Profile.Pronouns != nil {
			opts = append(opts, models.WithPronouns(*req.Profile.Pronouns))
			updatedFields = append(updatedFields, "profile.pronouns")
		}
		if req.Profile.Education != nil {
			opts = append(opts, models.WithEducation(*req.Profile.Education))
			updatedFields = append(updatedFields, "profile.education")
		}
		if req.Profile.Skills != nil {
			opts = append(opts, models.WithSkills(strings.Split(*req.Profile.Skills, ",")))
			updatedFields = append(updatedFields, "profile.skills")
		}
		if req.Profile.Interests != nil {
			opts = append(opts, models.WithInterests(strings.Split(*req.Profile.Interests, ",")))
			updatedFields = append(updatedFields, "profile.interests")
		}
	}
	if req.Settings != nil {
		if req.Settings.BrandColor != nil {
			opts = append(opts, models.WithBrandColor(*req.Settings.BrandColor))
			updatedFields = append(updatedFields, "settings.brand_color")
		}
	}

	if len(opts) == 0 {
		Logger.Info(c.Context()).WithFields("user_id", userID).Logs("No fields provided for update")
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "No changes provided",
			"status":  fiber.StatusOK,
			"user":    user,
		})
	}

	updatedUser, err := models.UpdateUser(c.Context(), Redis, DB, userID, opts...)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Failed to update user")
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update profile"})
	}

	Redis.Del(c.Context(), userKey)
	userJSON, _ := json.Marshal(updatedUser)
	if err := Redis.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Failed to update Redis cache")
	}

	Logger.Info(c.Context()).WithFields("user_id", userID).Logs("User profile updated successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Section updated successfully",
		"status":  fiber.StatusOK,
		"user":    updatedUser,
	})
}

func getBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

// UpdateUserNotificationPrefrences updates the user's notification preferences
func UpdateUserNotificationPrefrences(c *fiber.Ctx) error {
	type UpdateData struct {
		EmailOnLikes    *bool `json:"email_on_likes" validate:"omitempty"`
		EmailOnComments *bool `json:"email_on_comments" validate:"omitempty"`
		EmailOnMentions *bool `json:"email_on_mentions" validate:"omitempty"`
		EmailOnFollower *bool `json:"email_on_followers" validate:"omitempty"`
		EmailOnBadge    *bool `json:"email_on_badge" validate:"omitempty"`
		EmailOnUnread   *bool `json:"email_on_unread" validate:"omitempty"`
		EmailOnNewPosts *bool `json:"email_on_new_posts" validate:"omitempty"`
	}

	userIDRaw, ok := c.Locals("user_id").(string)
	if !ok || userIDRaw == "" {
		Logger.Warn(c.Context()).Logs("UpdateUserProfile attempted without user ID in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	userID, err := uuid.Parse(userIDRaw)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "userID", userIDRaw).Logs("Invalid user ID format in UpdateUserProfile")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	allowed := RateLimitting(c, userIDRaw, 1*time.Minute, 5, "profile_update_rate:")
	if !allowed {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  "Too many update attempts, try again later",
			"status": fiber.StatusTooManyRequests,
		})
	}

	var data UpdateData
	if err := utils.StrictBodyParser(c, &data); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Invalid request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	if err := Validator.Validate(data); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Validation failed")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	updatedUser, err := models.UpdateNotificationPreferences(
		c.Context(),
		Redis,
		DB,
		userID,
		getBool(data.EmailOnLikes),
		getBool(data.EmailOnComments),
		getBool(data.EmailOnMentions),
		getBool(data.EmailOnFollower),
		getBool(data.EmailOnBadge),
		getBool(data.EmailOnUnread),
		getBool(data.EmailOnNewPosts),
	)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Failed to update user")
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update profile"})
	}

	Logger.Info(c.Context()).WithFields("user_id", userID).Logs("User profile updated successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Section updated successfully",
		"status":  fiber.StatusOK,
		"user":    updatedUser,
	})
}

// UpdateUserCustomization updates the user's customiztion
func UpdateUserCustomization(c *fiber.Ctx) error {
	type UpdateData struct {
		ThemePreference *string `json:"theme_preference" validate:"omitempty,oneof=Light Dark"`
		BaseFont        *string `json:"base_font" validate:"omitempty,oneof=sans-serif sans jetbrainsmono hind-siliguri comic-sans"`
		SiteNavbar      *string `json:"site_navbar" validate:"omitempty,oneof=fixed static"`
		ContentEditor   *string `json:"content_editor" validate:"omitempty,oneof=rich basic"`
		ContentMode     *int    `json:"content_mode" validate:"omitempty,oneof=1 2 3 4 5"`
	}

	userIDRaw, ok := c.Locals("user_id").(string)
	if !ok || userIDRaw == "" {
		Logger.Warn(c.Context()).Logs("UpdateUserProfile attempted without user ID in context")
	}

	userID, err := uuid.Parse(userIDRaw)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "userID", userIDRaw).Logs("Invalid user ID format in UpdateUserProfile")
	}

	allowed := RateLimitting(c, userIDRaw, 1*time.Minute, 5, "profile_update_rate:")
	if !allowed {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  "Too many update attempts, try again later",
			"status": fiber.StatusTooManyRequests,
		})
	}

	var data UpdateData
	if err := utils.StrictBodyParser(c, &data); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Invalid request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	if err := Validator.Validate(data); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Validation failed")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	userKey := "user:" + userIDRaw
	var user *models.User
	cachedUser, err := Redis.Get(c.Context(), userKey).Result()
	if err == nil {
		user = &models.User{}
		if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
			Logger.Warn(c.Context()).WithFields("error", err, "userID", userIDRaw).Logs("Failed to unmarshal cached user from Redis")
			user = nil
		}
	}

	var opts []models.UserOption
	updatedFields := []string{}
	if data.ThemePreference != nil {
		opts = append(opts, models.WithThemePreference(*data.ThemePreference))
		updatedFields = append(updatedFields, "settings.theme_preference")
	}
	if data.BaseFont != nil {
		opts = append(opts, models.WithBaseFont(*data.BaseFont))
		updatedFields = append(updatedFields, "settings.base_font")
	}
	if data.SiteNavbar != nil {
		opts = append(opts, models.WithSiteNavbar(*data.SiteNavbar))
		updatedFields = append(updatedFields, "settings.site_navbar")
	}
	if data.ContentEditor != nil {
		opts = append(opts, models.WithContentEditor(*data.ContentEditor))
		updatedFields = append(updatedFields, "settings.content_editor")
	}
	if data.ContentMode != nil {
		opts = append(opts, models.WithContentMode(*data.ContentMode))
		updatedFields = append(updatedFields, "settings.content_mode")
	}

	if len(opts) == 0 {
		Logger.Info(c.Context()).WithFields("user_id", userID).Logs("No fields provided for update")
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "No changes provided",
			"status":  fiber.StatusOK,
			"user":    user,
		})
	}

	updatedUser, err := models.UpdateUser(c.Context(), Redis, DB, userID, opts...)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Failed to update user")
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update profile"})
	}

	Redis.Del(c.Context(), userKey)
	userJSON, _ := json.Marshal(updatedUser)
	if err := Redis.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Failed to update Redis cache")
	}

	Logger.Info(c.Context()).WithFields("user_id", userID).Logs("User profile updated successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Section updated successfully",
		"status":  fiber.StatusOK,
		"user":    updatedUser,
	})
}

// UpdateUserAccount updates the user's account
func UpdateUserAccount(c *fiber.Ctx) error {
	type UpdatePasswordRequest struct {
		CurrentPassword string `json:"current_password" validate:"required,min=6"`
		NewPassword     string `json:"new_password" validate:"required,min=6"`
		ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
	}

	userIDRaw, ok := c.Locals("user_id").(string)
	if !ok || userIDRaw == "" {
		Logger.Warn(c.Context()).Logs("UpdateUserPassword attempted without user ID in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	userID, err := uuid.Parse(userIDRaw)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "userID", userIDRaw).Logs("Invalid user ID format in UpdateUserPassword")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	allowed := RateLimitting(c, userIDRaw, 5*time.Minute, 5, "rate:change-password:")
	if !allowed {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  "Too many update attempts, try again later",
			"status": fiber.StatusTooManyRequests,
		})
	}

	var req UpdatePasswordRequest
	if err := utils.StrictBodyParser(c, &req); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Invalid request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	if err := Validator.Validate(req); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Validation failed")
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

	userKey := "user:" + userID.String()
	var user *models.User
	cachedUser, err := Redis.Get(c.Context(), userKey).Result()
	if err == nil {
		user = &models.User{}
		if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
			Logger.Warn(c.Context()).WithFields("error", err, "userID", userID.String()).Logs("Failed to unmarshal cached user from Redis")
			user = nil
		}
	}

	if err := utils.ComparePasswords(user.Password, req.CurrentPassword); err != nil {
		Logger.Warn(c.Context()).WithFields("email", user.Email).Logs("Invalid password provided")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Invalid current password",
			"status": fiber.StatusUnauthorized,
		})
	}

	if !utils.IsStrongPassword(req.NewPassword) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "new password does not meet strength requirements",
			"status":  fiber.StatusBadRequest,
		})
	}

	if req.CurrentPassword == req.NewPassword {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message": "new password cannot be the same as current",
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

	_, err = models.UpdateUser(
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

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	Redis.Del(c.Context(), userKey)

	c.Set("Authorization", "")
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	c.Set("Pragma", "no-cache")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "DENY")
	c.Set("Content-Security-Policy", "default-src 'self'")

	Logger.Info(c.Context()).WithFields("user_id", userID).Logs("User profile updated successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Section updated successfully. Please log in again.",
		"status":  fiber.StatusOK,
	})
}

// DeleteUserAccount deletes the authenticated user's account with Redis optimization
func DeleteUserAccount(c *fiber.Ctx) error {
	type ConfirmData struct {
		Confirm bool `json:"confirm" validate:"required"`
	}

	userIDRaw, ok := c.Locals("user_id").(string)
	if !ok || userIDRaw == "" {
		Logger.Warn(c.Context()).Logs("UpdateUserPassword attempted without user ID in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	userID, err := uuid.Parse(userIDRaw)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "userID", userIDRaw).Logs("Invalid user ID format in UpdateUserPassword")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	allowed := RateLimitting(c, userIDRaw, 5*time.Minute, 5, "rate:delete-user:")
	if !allowed {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  "Too many update attempts, try again later",
			"status": fiber.StatusTooManyRequests,
		})
	}

	var req ConfirmData
	if err := utils.StrictBodyParser(c, &req); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Invalid request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	if err := Validator.Validate(req); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Validation failed")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	userKey := "user:id:" + userID.String()
	exists, err := Redis.Exists(c.Context(), userKey).Result()
	if err == nil {
		Logger.Warn(c.Context()).WithFields("error", err, "userID", userID.String()).Logs("Failed to check user existence in Redis")
	}

	if exists == 0 {
		user, err := models.GetUserBy(c.Context(), Redis, DB, "id = ?", []interface{}{userID})
		if err != nil {
			Logger.Error(c.Context()).WithFields("error", err, "userID", userID).Logs("Database error while fetching user profile")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to fetch user profile",
				"status": fiber.StatusInternalServerError,
			})
		}

		userJSON, _ := json.Marshal(user)
		if err := Redis.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
			Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Failed to update Redis cache")
		}
	}

	err = models.DeleteUser(c.Context(), Redis, DB, userID)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).WithFields("user_id", userID).Logs("Failed to delete user")
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete user"})
	}
	Redis.Del(c.Context(), userKey)

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	c.Set("Authorization", "")
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	c.Set("Pragma", "no-cache")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "DENY")
	c.Set("Content-Security-Policy", "default-src 'self'")
	c.ClearCookie("access_token")
	c.ClearCookie("refresh_token")
	c.Locals("user_id", nil)

	Logger.Info(c.Context()).WithFields("user_id", userID).Logs("User account deleted successfully")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Account deleted successfully",
		"status":  fiber.StatusOK,
	})
}

// FollowUser follows a user
func FollowUser(c *fiber.Ctx) error {
	username := c.Params("username")
	if username == "" {
		Logger.Warn(c.Context()).Logs("FollowUser attempted without username in URL")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Username is required",
			"status": fiber.StatusBadRequest,
		})
	}
	if len(username) > 20 {
		Logger.Warn(c.Context()).Logs("FollowUser attempted with username exceeding length limit")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Username exceeds length limit",
			"status": fiber.StatusBadRequest,
		})
	}
	if strings.Contains(username, " ") {
		Logger.Warn(c.Context()).Logs("FollowUser attempted with username containing spaces")
		username = strings.ReplaceAll(username, " ", "")
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

	err = followU.FollowUser(c.Context(), Redis, DB, username)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			Logger.Warn(c.Context()).WithFields("error", err, "following_username", username).Logs("Target user not found")
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":  "Target user not found",
				"status": fiber.StatusNotFound,
			})
		}
		// Check if the error is due to already following (GORM doesn't return a specific error for this, so we need to check the association)
		if err := DB.WithContext(c.Context()).Model(&followU).Association("Following").Find(&followU.Following); err == nil {
			for _, f := range followU.Following {
				if f.Username == username {
					Logger.Info(c.Context()).WithFields("user_id", followerID, "following_username", username).Logs("User already following target")
					return c.Status(fiber.StatusOK).JSON(fiber.Map{
						"message": "Already following this user",
						"status":  fiber.StatusOK,
					})
				}
			}
		}
		Logger.Error(c.Context()).WithFields("error", err.Error(), "user_id", followerID, "following_username", username).Logs("Failed to follow user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  err.Error(),
			"status": fiber.StatusInternalServerError,
		})
	}

	followerKey := "user:" + followerID.String()
	followingKey := "user:" + username
	Redis.Del(c.Context(), followerKey, followingKey)

	Logger.Info(c.Context()).WithFields("user_id", followerID, "following_username", username).Logs("User followed successfully")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User followed successfully",
		"status":  fiber.StatusOK,
	})
}

// UnfollowUser unfollows a user
func UnfollowUser(c *fiber.Ctx) error {
	username := c.Params("username")
	if username == "" {
		Logger.Warn(c.Context()).Logs("FollowUser attempted without username in URL")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Username is required",
			"status": fiber.StatusBadRequest,
		})
	}
	if len(username) > 20 {
		Logger.Warn(c.Context()).Logs("FollowUser attempted with username exceeding length limit")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Username exceeds length limit",
			"status": fiber.StatusBadRequest,
		})
	}
	if strings.Contains(username, " ") {
		Logger.Warn(c.Context()).Logs("FollowUser attempted with username containing spaces")
		username = strings.ReplaceAll(username, " ", "")
	}
	userIDRaw, ok := c.Locals("user_id").(string)
	if !ok || userIDRaw == "" {
		Logger.Warn(c.Context()).Logs("UnfollowUser attempted without user_id in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	followerID, err := uuid.Parse(userIDRaw)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "user_id", userIDRaw).Logs("Invalid user_id format in UnfollowUser")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
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

	err = followU.UnfollowUser(c.Context(), Redis, DB, username)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			Logger.Warn(c.Context()).WithFields("error", err, "following_username", username).Logs("Target user not found")
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":  "Target user not found",
				"status": fiber.StatusNotFound,
			})
		}
		Logger.Error(c.Context()).WithFields("error", err, "user_id", followerID, "following_username", username).Logs("Failed to unfollow user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to unfollow user",
			"status": fiber.StatusInternalServerError,
		})
	}

	// Check if the user was actually following (since the helper doesn't explicitly tell us)
	if err := DB.WithContext(c.Context()).Model(&followU).Association("Following").Find(&followU.Following); err == nil {
		isFollowing := false
		for _, f := range followU.Following {
			if f.Username == username {
				isFollowing = true
				break
			}
		}
		if !isFollowing {
			Logger.Info(c.Context()).WithFields("user_id", followerID, "following_username", username).Logs("User unfollowed successfully")
			following, _ := models.GetUserBy(c.Context(), Redis, DB, "username = ?", []interface{}{username})
			followingKey := "user:" + username
			publicFollowerKey := "public_user:" + followU.Username
			publicFollowingKey := "public_user:" + following.Username
			Redis.Del(c.Context(), followingKey, publicFollowerKey, publicFollowingKey)
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"message": "User unfollowed successfully",
				"status":  fiber.StatusOK,
			})
		}
	}

	Logger.Info(c.Context()).WithFields("user_id", followerID, "following_username", username).Logs("User unfollowed successfully")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User unfollowed successfully",
		"status":  fiber.StatusOK,
	})
}

// GetUserNotifications returns user notifications
func GetUserNotifications(c *fiber.Ctx) error {
	userIDRaw, ok := c.Locals("user_id").(string)
	if !ok || userIDRaw == "" {
		Logger.Warn(c.Context()).Logs("GetUserNotifications attempted without user_id in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	userID, err := uuid.Parse(userIDRaw)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "userID", userIDRaw).Logs("Invalid user ID format in GetUserNotifications")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	cacheKey := "user:" + userIDRaw
	cachedNotifications, err := Redis.Get(c.Context(), cacheKey).Result()
	if err == nil {
		var publicUser models.User
		if err := json.Unmarshal([]byte(cachedNotifications), &publicUser); err == nil {
			Logger.Info(c.Context()).WithFields("user_id", userIDRaw).Logs("user notifications served from cache")
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"message":       "User notifications retrieved successfully",
				"status":        fiber.StatusOK,
				"notifications": publicUser.Notifications,
			})
		}
		Logger.Warn(c.Context()).WithFields("error", err, "user_id", userIDRaw).Logs("Failed to unmarshal cached user notifications")
	}
	user, err := models.GetUserBy(c.Context(), Redis, DB, "id = ?", []interface{}{userID}, "")
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).WithFields("user_id", userIDRaw).Logs("Failed to fetch user by id")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "User not found",
			"status": fiber.StatusNotFound,
		})
	}

	if len(user.Notifications) == 0 {
		Logger.Info(c.Context()).WithFields("user_id", userIDRaw).Logs("No notifications found for user")
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":       "No notifications found",
			"status":        fiber.StatusOK,
			"notifications": []string{},
		})
	}
	notificationsJSON, _ := json.Marshal(user.Notifications)
	if err := Redis.Set(c.Context(), cacheKey, notificationsJSON, 5*time.Minute).Err(); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userIDRaw).Logs("Failed to cache user notifications")
	}
	Logger.Info(c.Context()).WithFields("user_id", userIDRaw).Logs("user notifications retrieved successfully")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":       "Public notifications retrieved successfully",
		"status":        fiber.StatusOK,
		"notifications": user.Notifications,
	})
}

// GetUserNotificationID returns user notification by ID
func GetUserNotificationID(c *fiber.Ctx) error {
	userIDRaw, ok := c.Locals("user_id").(string)
	if !ok || userIDRaw == "" {
		Logger.Warn(c.Context()).Logs("GetUserNotificationID attempted without user_id in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	notificationID := c.Params("notificationId")
	if notificationID == "" {
		Logger.Warn(c.Context()).Logs("GetUserNotificationID attempted without notification_id in URL")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Notification ID is required",
			"status": fiber.StatusBadRequest,
		})
	}

	notiID, err := uuid.Parse(notificationID)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err, "notification_id", notificationID).Logs("Invalid notification ID format in GetUserNotificationID")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid notification ID",
			"status": fiber.StatusBadRequest,
		})
	}

	cacheKey := "user:" + userIDRaw
	cachedNotifications, err := Redis.Get(c.Context(), cacheKey).Result()
	if err == nil {
		var publicUser models.User
		if err := json.Unmarshal([]byte(cachedNotifications), &publicUser); err == nil {
			Logger.Info(c.Context()).WithFields("user_id", userIDRaw).Logs("user notifications served from cache")
			var k models.Notification
			for v, notification := range publicUser.Notifications {
				if notification.ID == notiID {
					k = publicUser.Notifications[v]
					break
				}
			}
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"message":      "User notifications retrieved successfully",
				"status":       fiber.StatusOK,
				"notification": k,
			})
		}
		Logger.Warn(c.Context()).WithFields("error", err, "user_id", userIDRaw).Logs("Failed to unmarshal cached user notifications")
	}

	if strings.Contains(notificationID, " ") {
		Logger.Warn(c.Context()).Logs("GetUserNotificationID attempted with notification_id containing spaces")
		notificationID = strings.ReplaceAll(notificationID, " ", "")
	}
	notificationid, _ := uuid.Parse(notificationID)
	notification, err := models.GetNotification(c.Context(), Redis, DB, notificationid)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).WithFields("user_id", userIDRaw).Logs("Failed to fetch user notification")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "Notification not found",
			"status": fiber.StatusNotFound,
		})
	}
	notificationJSON, _ := json.Marshal(notification)
	if err := Redis.Set(c.Context(), cacheKey, notificationJSON, 5*time.Minute).Err(); err != nil {
		Logger.Warn(c.Context()).WithFields("error", err).WithFields("user_id", userIDRaw).Logs("Failed to cache user notification")
	}
	Logger.Info(c.Context()).WithFields("user_id", userIDRaw).Logs("user notification retrieved successfully")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":      "User notification retrieved successfully",
		"status":       fiber.StatusOK,
		"notification": notification,
	})
}
