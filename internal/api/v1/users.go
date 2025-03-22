package v1

import (
	"encoding/json"
	"fmt"
	"strconv"
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

func Register(c *fiber.Ctx) error {
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

	ui.Email = strings.ToLower(strings.TrimSpace(ui.Email))

	hashedPass, err := utils.HashPassword(ui.Password)
	if err != nil {
		Logger.Error(c.Context()).WithFields("error", err).Logs(fmt.Sprintf("Failed to hash password: %v", err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

	gotp, err := utils.GenerateOTP()
	if err != nil {
		Logger.Error(c.Context()).Logs(fmt.Sprintf("Failed to generate OTP: %v", err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate activation code",
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
	otp, err := utils.HashPassword(fmt.Sprintf("%d", gotp))
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
		OTP int64 `json:"otp" validate:"required"`
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

	if err := utils.ComparePasswords(otpHash, strconv.FormatInt(ar.OTP, 10)); err != nil {
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
		// Return a 400 Bad Request response for invalid user ID
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid user ID",
			"status": fiber.StatusBadRequest,
		})
	}

	userKey := "user:" + uid.String()
	// Declare a variable to hold the user struct
	var user *models.User
	// Attempt to fetch the user profile from Redis
	cachedUser, err := Redis.Get(c.Context(), userKey).Result()
	// Check if the user profile is cached in Redis
	if err == nil {
		// Unmarshal the cached JSON into a user struct
		user = &models.User{}
		if err := json.Unmarshal([]byte(cachedUser), user); err != nil {
			// Log a warning if unmarshaling fails (fallback to DB)
			Logger.Warn(c.Context()).WithFields("error", err, "userID", uid).Logs("Failed to unmarshal cached user from Redis")
			user = nil
		}
	}
	// Check if the user wasn’t found in Redis or unmarshaling failed
	if err == redis.Nil || user == nil {
		// Fetch the user profile from the database by ID
		user, err = models.GetUserBy(c.Context(), Redis, DB, "id = ?", []interface{}{uid}, "")
		// Check if fetching the user failed
		if err != nil {
			// Log an error with details if the user isn’t found or DB fails
			Logger.Error(c.Context()).WithFields("error", err, "userID", uid).Logs("Database error while fetching user profile")
			// Return a 500 Internal Server Error if DB query fails
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to fetch user profile",
				"status": fiber.StatusInternalServerError,
			})
		}
		// Marshal the user to JSON for caching
		userJSON, err := json.Marshal(user)
		// Check if marshaling failed
		if err != nil {
			// Log a warning if marshaling fails (non-critical, proceed)
			Logger.Warn(c.Context()).WithFields("error", err, "userID", uid).Logs("Failed to marshal user for Redis caching")
		} else {
			// Cache the user profile in Redis with a 30-minute TTL
			if err := Redis.Set(c.Context(), userKey, userJSON, 30*time.Minute).Err(); err != nil {
				// Log a warning if caching fails (non-critical)
				Logger.Warn(c.Context()).WithFields("error", err, "userID", uid).Logs("Failed to cache user profile in Redis")
			}
		}
	} else if err != nil {
		// Log an error if Redis fails for another reason (proceed with DB fallback)
		Logger.Error(c.Context()).WithFields("error", err, "userID", uid).Logs("Redis error fetching user profile")
		// Fetch from DB as a fallback
		user, err = models.GetUserBy(c.Context(), Redis, DB, "id = ?", []interface{}{uid})
		if err != nil {
			Logger.Error(c.Context()).WithFields("error", err, "userID", uid).Logs("Database error while fetching user profile")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "Failed to fetch user profile",
				"status": fiber.StatusInternalServerError,
			})
		}
	}

	// Prepare user profile response with selective fields
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
	}

	// Log successful profile retrieval
	Logger.Info(c.Context()).WithFields("userID", uid).Logs("User profile retrieved successfully")
	// Return a 200 OK response with the user profile
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Profile retrieved successfully",
		"status":  fiber.StatusOK,
		"user":    profileResponse,
	})
}
