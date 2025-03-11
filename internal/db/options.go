package db

import (
	"time"

	"github.com/mnuddindev/devpulse/pkg/logger"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// WithLogger defines a function to configure DB connection.
func WithLogger(logger *logger.Logger) DBOptions {
	return func(db *gorm.DB) error {
		db.Config.Logger = gormLogger.New(
			logger.Log,
			gormLogger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  gormLogger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)
		return nil
	}
}
