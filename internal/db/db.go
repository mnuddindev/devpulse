package db

import (
	"context"
	"sync"

	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DBInstance *gorm.DB
	Once       sync.Once
	DBMu       sync.Mutex
)

type DBOptions func(*gorm.DB) error

func NewDB(ctx context.Context, dsn string, models []interface{}, opts ...DBOptions) (*gorm.DB, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var InitErr error
	Once.Do(func() {
		if err := ctx.Err(); err != nil {
			InitErr = utils.WrapError(err, utils.ErrInternalServerError.Code, "DB initialization canceled")
			return
		}

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			InitErr = utils.NewError(utils.ErrInternalServerError.Code, "Failed to connect to Database", err.Error())
			return
		}

		for _, opt := range opts {
			if err := opt(db); err != nil {
				InitErr = utils.NewError(utils.ErrForbidden.Code, "Failed to apply DB Options", err.Error())
				return
			}
		}

		select {
		case <-ctx.Done():
			InitErr = utils.WrapError(ctx.Err(), utils.ErrInternalServerError.Code, "db migration canceled")
			return
		default:
			if err := db.WithContext(ctx).AutoMigrate(models...); err != nil {
				InitErr = utils.NewError(utils.ErrInternalServerError.Code, "Failed to Migrate models", err.Error())
				return
			}
		}

		DBMu.Lock()
		DBInstance = db
		DBMu.Unlock()
	})

	if InitErr != nil {
		return nil, InitErr
	}

	if DBInstance == nil {
		return nil, utils.NewError(utils.ErrInternalServerError.Code, "Database not initialized")
	}

	return DBInstance, nil
}

func GetDB() *gorm.DB {
	DBMu.Lock()
	defer DBMu.Unlock()

	if DBInstance == nil {
		panic("Database connection not initialized; call NewDB first")
	}
	return DBInstance
}

func CloseDB(logger *logger.Logger) error {
	DBMu.Lock()
	defer DBMu.Unlock()

	if DBInstance == nil {
		return nil
	}

	sqlDB, err := DBInstance.DB()
	if err != nil {
		logger.Error(context.Background()).WithMeta(utils.Map{"error": err.Error()}).Logs("Failed to get DB handle for closing")
		return utils.NewError(utils.ErrInternalServerError.Code, "Failed to close database", err.Error())
	}

	if err := sqlDB.Close(); err != nil {
		logger.Error(context.Background()).WithMeta(utils.Map{"error": err.Error()}).Logs("PostgreSQL database close failed")
		return utils.NewError(utils.ErrInternalServerError.Code, "Failed to close database", err.Error())
	}
	logger.Info(context.Background()).Logs("PostgreSQL database connection closed successfully")
	DBInstance = nil
	return nil
}
