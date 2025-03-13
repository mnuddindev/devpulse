package db

import (
	"context"

	"github.com/mnuddindev/devpulse/internal/models"
	"github.com/mnuddindev/devpulse/pkg/logger"
	storage "github.com/mnuddindev/devpulse/pkg/redis"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DBInstance *gorm.DB
)

func NewDB(ctx context.Context, dsn string, rclient *storage.RedisClient, log *logger.Logger) (*gorm.DB, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "DB initialization canceled")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, utils.NewError(utils.ErrInternalServerError.Code, "Failed to connect to Database", err.Error())
	}

	DBInstance = db
	if DBInstance == nil {
		return nil, utils.NewError(utils.ErrInternalServerError.Code, "Database not initialized")
	}

	if err := db.WithContext(ctx).AutoMigrate(models.RegisterModels()...); err != nil {
		return nil, utils.NewError(utils.ErrInternalServerError.Code, "Failed to auto-migrate models", err.Error())
	}

	if err := models.SeedRoles(ctx, db, rclient, log); err != nil {
		return nil, err
	}

	return DBInstance, nil
}

func GetDB() *gorm.DB {
	if DBInstance == nil {
		panic("Database connection not initialized; call NewDB first")
	}
	return DBInstance
}

func CloseDB() error {
	if DBInstance == nil {
		return nil
	}

	sqlDB, err := DBInstance.DB()
	if err != nil {
		return utils.NewError(utils.ErrInternalServerError.Code, "Failed to get DB handle for closing", err.Error())
	}

	if err := sqlDB.Close(); err != nil {
		return utils.NewError(utils.ErrInternalServerError.Code, "Failed to close PostgreSQL database", err.Error())
	}

	DBInstance = nil
	return nil
}
