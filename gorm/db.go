package gorm

import (
	"fmt"

	cfg "github.com/mnuddindev/devpulse/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(co *cfg.Postgres) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", co.Host, co.User, co.Name, co.Pass)
	client, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("Error while connecting to database: %v\n", err)
		return nil
	}
	fmt.Println("Connected to database")
	return client
}
