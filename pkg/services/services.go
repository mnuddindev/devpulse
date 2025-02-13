package services

import (
	"github.com/mnuddindev/devpulse/gorm"
	grm "gorm.io/gorm"
)

// UserSystem struct that holds a reference to the CRUD operations using Gorm.
type UserSystem struct {
	crud *gorm.GormDB
}

// NewUserSystem initializes a new UserSystem with a given database connection.
func NewUserSystem(db *grm.DB) *UserSystem {
	return &UserSystem{
		crud: gorm.NewGormDB(db),
	}
}
