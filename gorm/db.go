package gorm

import (
	"fmt"

	cfg "github.com/mnuddindev/devpulse/config"
	"github.com/mnuddindev/devpulse/pkg/models"
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
		fmt.Printf("Error while connecting to database: %v\n", err)
		return nil
	}
	// Print a message if the connection is successful
	fmt.Println("Connected to database")

	// Migrate the schema
	client.AutoMigrate(&models.User{})

	// Return the database connection
	return client
}
