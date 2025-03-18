package auth

import (
	"github.com/mnuddindev/devpulse/pkg/logger"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"gorm.io/gorm"
)

type Options struct {
	DB      *gorm.DB
	Rclient *storage.RedisClient
	Logger  *logger.Logger
}
