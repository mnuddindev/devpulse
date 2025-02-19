package services

import (
	"github.com/mnuddindev/devpulse/pkg/services/users"
	"gorm.io/gorm"
)

// UserSystem struct that holds a reference to the CRUD operations using Gorm.
type Systems struct {
	*users.UserSystem
}

// NewUserSystem initializes a new UserSystem with a given database connection.
func NewSystem(db *gorm.DB) *Systems {
	return &Systems{
		UserSystem: users.NewUserSystem(db),
	}
}
