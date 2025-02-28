package gorm

import (
	"fmt"

	cfg "github.com/mnuddindev/devpulse/config"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(co *cfg.Postgres) *gorm.DB {
	// Connection credentials for the database
	dsn := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", co.Host, co.User, co.Name, co.Pass)
	// Connect to the database
	client, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	// Check if there is an error while connecting to the database
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Error while connecting to database")
		return nil
	}
	// Print a message if the connection is successful
	logger.Log.Info("Connected to database")

	if err := client.Debug().AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Notification{},
		&models.NotificationPrefrences{},
		&models.Posts{},
		&models.Comment{},
		&models.Reaction{},
		&models.Bookmark{},
		&models.PostAnalytics{},
		&models.Series{},
		&models.SeriesAnalytics{},
		&models.Collection{},
		&models.CommentFlag{},
		&models.SocialMediaPreview{},
		&models.Tag{},
		&models.TagAnalytics{},
	); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Error while migrating the schema for a specific model")
		return nil
	}

	logger.Log.Info("Schema auto migrated successfully")

	SeedRoles(client)
	// SeedBadges(client)

	// Return the database connection
	return client
}

func SeedRoles(db *gorm.DB) {
	// Define permissions
	permissions := []models.Permission{
		// Post-related permissions
		{Name: "create_post"},     // Create a new post
		{Name: "delete_any_post"}, // Delete any post (admin/moderator)
		{Name: "delete_own_post"}, // Delete own posts
		{Name: "edit_any_post"},   // Edit any post (admin/moderator)
		{Name: "edit_own_post"},   // Edit own posts
		{Name: "feature_posts"},   // Mark posts as featured (moderator/admin)
		{Name: "moderate_post"},   // Moderate posts (e.g., approve, flag)
		{Name: "read_post"},       // View posts (basic access)

		// Comment-related permissions
		{Name: "create_comment"},     // Add a comment to a post
		{Name: "delete_any_comment"}, // Delete any comment (admin/moderator)
		{Name: "delete_own_comment"}, // Delete own comments
		{Name: "edit_any_comment"},   // Edit any comment (admin/moderator)
		{Name: "edit_own_comment"},   // Edit own comments
		{Name: "moderate_comment"},   // Moderate comments (e.g., approve, flag)

		// User-related permissions
		{Name: "ban_user"},           // Ban a user from the platform (admin)
		{Name: "create_user"},        // Register new users (admin or system-level)
		{Name: "delete_any_user"},    // Delete any user account (admin)
		{Name: "delete_own_profile"}, // Delete own account
		{Name: "edit_any_user"},      // Edit any user’s profile/data (admin/moderator)
		{Name: "edit_own_profile"},   // Edit own profile
		{Name: "follow_user"},        // Follow another user
		{Name: "moderate_user"},      // Moderate user accounts (e.g., warn, restrict)
		{Name: "read_user_profile"},  // View other users’ profiles (new)
		{Name: "unfollow_user"},      // Unfollow another user

		// Role and permission management
		{Name: "assign_roles"}, // Assign roles to users (admin/moderator)
		{Name: "create_roles"}, // Create new roles (admin)
		{Name: "delete_roles"}, // Delete roles (admin)
		{Name: "edit_roles"},   // Edit role details/permissions (admin)
		{Name: "manage_roles"}, // Broad permission for role management (admin)

		// Reaction-related permissions
		{Name: "create_reaction"},     // Add a reaction (e.g., like, heart) to posts/comments
		{Name: "delete_any_reaction"}, // Delete any reaction (admin/moderator)
		{Name: "delete_own_reaction"}, // Delete own reactions
		{Name: "edit_any_reaction"},   // Edit any reaction (admin/moderator, rare use case)
		{Name: "edit_own_reaction"},   // Edit own reactions

		// Tag-related permissions
		{Name: "create_tag"},   // Create a new tag
		{Name: "delete_tag"},   // Delete a tag (admin/moderator)
		{Name: "edit_tag"},     // Edit a tag (admin/moderator)
		{Name: "follow_tag"},   // Follow a tag
		{Name: "moderate_tag"}, // Moderate tags (e.g., approve, merge)
		{Name: "unfollow_tag"}, // Unfollow a tag

		// Site-wide and moderation permissions
		{Name: "give_suggestion"},      // Submit suggestions for site improvements
		{Name: "manage_analytics"},     // View/use site analytics (admin, new)
		{Name: "manage_notifications"}, // Manage notification settings for all users (admin, new)
		{Name: "manage_site_settings"}, // Manage global site settings (admin, replaces "site_setting")
		{Name: "need_moderation"},      // Flag content/users needing moderation (member feature)
		{Name: "report_content"},       // Report posts/comments/users (new, for all users)
	}
	for _, perm := range permissions {
		db.FirstOrCreate(&perm, models.Permission{Name: perm.Name})
	}

	// Define roles with permissions
	roles := []struct {
		Name        string
		Permissions []string
	}{
		{"member", []string{
			"read_post",
			"create_comment",
			"edit_own_comment",
			"delete_own_comment",
			"give_reaction",
			"follow_tag",
			"unfollow_tag",
			"follow_user",
			"unfollow_user",
			"give_reaction",
			"delete_own_profile",
			"need_moderation",
			"edit_own_profile",
		}},
		{"author", []string{
			"read_post",
			"create_post",
			"edit_own_post",
			"delete_own_post",
			"create_comment",
			"edit_own_comment",
			"delete_own_comment",
			"give_reaction",
			"follow_tag",
			"unfollow_tag",
			"follow_user",
			"unfollow_user",
			"give_reaction",
			"delete_own_profile",
			"edit_own_profile",
		}},
		{"trusted_member", []string{
			"read_post",
			"create_post",
			"edit_own_post",
			"delete_own_post",
			"create_comment",
			"edit_own_comment",
			"delete_own_comment",
			"give_reaction",
			"follow_tag",
			"unfollow_tag",
			"follow_user",
			"unfollow_user",
			"give_reaction",
			"delete_own_profile",
			"give_suggestion",
			"edit_own_profile",
		}},
		{"tag_moderator", []string{
			"read_post",
			"create_post",
			"edit_own_post",
			"delete_own_post",
			"create_comment",
			"edit_own_comment",
			"delete_own_comment",
			"give_reaction",
			"follow_tag",
			"unfollow_tag",
			"follow_user",
			"unfollow_user",
			"give_reaction",
			"delete_own_profile",
			"give_suggestion",
			"edit_own_profile",
			"moderate_tag",
			"feature_posts",
			"ban_user",
		}},
		{"Moderator", []string{
			"read_post",
			"create_post",
			"edit_own_post",
			"delete_own_post",
			"edit_any_post",
			"create_comment",
			"delete_any_post",
			"moderate_post",
			"create_comment",
			"edit_own_comment",
			"delete_own_comment",
			"edit_any_comment",
			"moderate_comment",
			"create_user",
			"edit_user",
			"delete_user",
			"edit_own_profile",
			"delete_own_profile",
			"moderate_user",
			"ban_user",
			"follow_user",
			"unfollow_user",
			"create_roles",
			"edit_roles",
			"delete_roles",
			"create_reaction",
			"edit_reaction",
			"delete_reaction",
			"give_reaction",
			"create_tag",
			"edit_tag",
			"delete_tag",
			"moderate_tag",
			"follow_tag",
			"unfollow_tag",
			"need_moderation",
			"give_suggestion",
			"feature_posts",
		}},
		{"Admin", []string{"admin"}},
	}

	// Seed roles and permissions into the database
	for _, r := range roles {
		// Declare a variable to hold the role
		var role models.Role
		// Check if the role exists by name, create it if not
		if err := db.Where("name = ?", r.Name).First(&role).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create a new role with the given name
				role = models.Role{Name: r.Name}
				if err := db.Create(&role).Error; err != nil {
					logger.Log.WithFields(logrus.Fields{
						"error": err,
						"role":  r.Name,
					}).Error("Failed to create role")
					continue
				}
			} else {
				logger.Log.WithFields(logrus.Fields{
					"error": err,
					"role":  r.Name,
				}).Error("Database error fetching role")
				continue
			}
		}

		// Fetch or create permissions and associate them with the role
		var perms []models.Permission
		for _, permName := range r.Permissions {
			// Declare a variable to hold the permission
			var perm models.Permission
			// Check if the permission exists, create it if not
			if err := db.Where("name = ?", permName).First(&perm).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					perm = models.Permission{Name: permName}
					if err := db.Create(&perm).Error; err != nil {
						logger.Log.WithFields(logrus.Fields{
							"error": err,
							"perm":  permName,
						}).Error("Failed to create permission")
						continue
					}
				} else {
					logger.Log.WithFields(logrus.Fields{
						"error": err,
						"perm":  permName,
					}).Error("Database error fetching permission")
					continue
				}
			}
			// Add the permission to the list
			perms = append(perms, perm)
		}

		// Associate the permissions with the role
		if err := db.Model(&role).Association("Permissions").Replace(perms); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"error": err,
				"role":  r.Name,
			}).Error("Failed to associate permissions with role")
		}
	}
}

// func SeedBadges(db *gorm.DB) {
// 	badges := []string{
// 		"One Year Club",
// 		"Two Year Club",
// 		"Three Year Club",
// 		"Four Year Club",
// 		"Six Year Club",
// 		"Seven Year Club",
// 		"Eight Year Club",
// 		"Beloved Comment",
// 		"Warm Welcome",
// 		"Writing Debut",
// 		"Writing Streak",
// 		"Top 7",
// 		"Big Thread",
// 		"Fab 5",
// 	}

// 	image := []string{
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 		"https://localhost/badge.png",
// 	}

// 	for i, badge := range badges {
// 		var count int64
// 		db.Model(&models.Badge{}).Where("name = ?", badge).Count(&count)

// 		if count == 0 {
// 			db.Create(&models.Badge{Name: badge, Image: image[i]})
// 			fmt.Println("✅ Created badge:", badge)
// 		} else {
// 			return
// 		}
// 	}
// }
