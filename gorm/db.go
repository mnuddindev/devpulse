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

	// Migrate the schema
	if err = client.AutoMigrate(&models.User{}); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Error while migrating the schema")
		return nil
	}

	logger.Log.Info("Schema auto migrated successfully")

	// Return the database connection
	return client
}
