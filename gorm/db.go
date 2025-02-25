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
		{Name: "read_post"},
		{Name: "create_post"},
		{Name: "edit_own_post"},
		{Name: "delete_own_post"},
		{Name: "edit_any_post"},
		{Name: "delete_any_post"},
		{Name: "moderate_post"},
		{Name: "create_comment"},
		{Name: "edit_own_comment"},
		{Name: "delete_own_comment"},
		{Name: "edit_any_comment"},
		{Name: "moderate_comment"},
		{Name: "create_user"},
		{Name: "edit_user"},
		{Name: "delete_user"},
		{Name: "edit_own_profile"},
		{Name: "delete_own_profile"},
		{Name: "moderate_user"},
		{Name: "ban_user"},
		{Name: "follow_user"},
		{Name: "unfollow_user"},
		{Name: "create_roles"},
		{Name: "edit_roles"},
		{Name: "delete_roles"},
		{Name: "manage_roles"},
		{Name: "create_reaction"},
		{Name: "edit_reaction"},
		{Name: "delete_reaction"},
		{Name: "give_reaction"},
		{Name: "create_tag"},
		{Name: "edit_tag"},
		{Name: "delete_tag"},
		{Name: "moderate_tag"},
		{Name: "follow_tag"},
		{Name: "unfollow_tag"},
		{Name: "site_setting"},
		{Name: "need_moderation"},
		{Name: "give_suggestion"},
		{Name: "feature_posts"},
		{Name: "admin"},
	}
	for i := range permissions {
		db.FirstOrCreate(&permissions[i], models.Permission{Name: permissions[i].Name})
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

	for _, r := range roles {
		var role models.Role
		db.FirstOrCreate(&role, models.Role{Name: r.Name})
		var perms []models.Permission
		db.Find(&perms, "name IN ?", r.Permissions)
		db.Model(&role).Association("Permissions").Replace(perms)
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
