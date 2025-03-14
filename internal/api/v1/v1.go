package v1

import (
	"github.com/mnuddindev/devpulse/pkg/logger"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"gorm.io/gorm"
)

var (
	DB       *gorm.DB
	Redis    *storage.RedisClient
	Logger   *logger.Logger
	EmailCfg = utils.EmailConfig{
		SMTPHost:     "0.0.0.0",
		SMTPPort:     1025,
		SMTPUsername: "",
		SMTPPassword: "",
		AppURL:       "http://localhost:3000",
		FromEmail:    "no-reply@devpulse.com",
	}
	Validator = utils.NewValidator()
)
