package v1

import (
	"github.com/mnuddindev/devpulse/pkg/logger"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"gorm.io/gorm"
)

var (
	DB     *gorm.DB
	Redis  *storage.RedisClient
	Logger *logger.Logger
)
