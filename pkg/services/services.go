package services

import (
	"github.com/go-redis/redis/v8"
	postservices "github.com/mnuddindev/devpulse/pkg/services/posts"
	"github.com/mnuddindev/devpulse/pkg/services/users"
	"gorm.io/gorm"
)

// UserSystem struct that holds a reference to the CRUD operations using Gorm.
type Systems struct {
	*users.UserSystem
	*postservices.PostSystem
}

// NewUserSystem initializes a new UserSystem with a given database connection.
func NewSystem(db *gorm.DB, client *redis.Client) *Systems {
	return &Systems{
		UserSystem: users.NewUserSystem(db, client),
		PostSystem: postservices.NewPostSystem(db, client),
	}
}
