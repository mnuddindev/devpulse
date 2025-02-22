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

	models := []interface{}{
		&models.User{},
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
	}

	// Iterate over the model list and migrate only if the table doesn’t exist
	for _, model := range models {
		if !client.Migrator().HasTable(model) {
			if err := client.Debug().AutoMigrate(model); err != nil {
				logger.Log.WithFields(logrus.Fields{
					"error": err.Error(),
					"model": model, // Log the model causing the error
				}).Error("Error while migrating the schema for a specific model")
				return nil
			}
			logger.Log.WithFields(logrus.Fields{
				"model": model,
			}).Info("Table created successfully")
		} else {
			logger.Log.WithFields(logrus.Fields{
				"model": model,
			}).Info("Table already exists, skipping migration")
		}
	}

	logger.Log.Info("Schema auto migrated successfully")

	SeedRoles(client)
	SeedBadges(client)

	// Return the database connection
	return client
}

func SeedRoles(db *gorm.DB) {
	roles := []string{"member", "moderator", "author", "trusted_member", "admin"}

	for _, role := range roles {
		var count int64
		db.Model(&models.Role{}).Where("name = ?", role).Count(&count)

		if count == 0 {
			db.Create(&models.Role{Name: role})
			fmt.Println("✅ Created role:", role)
		} else {
			return
		}
	}
}

func SeedBadges(db *gorm.DB) {
	badges := []string{
		"One Year Club",
		"Two Year Club",
		"Three Year Club",
		"Four Year Club",
		"Six Year Club",
		"Seven Year Club",
		"Eight Year Club",
		"Beloved Comment",
		"Warm Welcome",
		"Writing Debut",
		"Writing Streak",
		"Top 7",
		"Big Thread",
		"Fab 5",
	}

	image := []string{
		"https://localhost/badge.png",
		"https://localhost/badge.png",
		"https://localhost/badge.png",
		"https://localhost/badge.png",
		"https://localhost/badge.png",
		"https://localhost/badge.png",
		"https://localhost/badge.png",
		"https://localhost/badge.png",
		"https://localhost/badge.png",
		"https://localhost/badge.png",
		"https://localhost/badge.png",
		"https://localhost/badge.png",
		"https://localhost/badge.png",
		"https://localhost/badge.png",
	}

	for i, badge := range badges {
		var count int64
		db.Model(&models.Badge{}).Where("name = ?", badge).Count(&count)

		if count == 0 {
			db.Create(&models.Badge{Name: badge, Image: image[i]})
			fmt.Println("✅ Created badge:", badge)
		} else {
			return
		}
	}
}
