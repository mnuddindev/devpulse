package gorm

import (
	"errors"

	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormDB struct
type GormDB struct {
	DB *gorm.DB
}

type CRUD interface {
	// Create a new record
	Create(model interface{}) error
	// Read a record by ID
	GetByID(model interface{}, id int) error
	// Get all records
	GetAll(model interface{}) error
	// GetByCondition records
	GetByCondition(model interface{}, condition string, args ...interface{}) error
	// Update a record by ID
	Update(model interface{}, condition string, args interface{}, updates interface{}) error
	// Delete a record by ID
	Delete(model interface{}, id int) error
}

// NewGormDB creates a new GormDB instance
func NewGormDB(db *gorm.DB) *GormDB {
	return &GormDB{DB: db}
}

// Create a new record
func (g *GormDB) Create(model interface{}) error {
	if err := g.DB.Create(model).Error; err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"model": model,
		}).Error("Error while creating a new record")
		return errors.New("error while creating a new record")
	}
	logger.Log.WithFields(logrus.Fields{
		"model": model,
	}).Info("Record Created Successfully!!")
	return nil
}

// GetByID a record by ID
func (g *GormDB) GetByID(model interface{}, id string) error {
	if err := g.DB.First(model, id).Error; err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"id":    id,
		}).Error("Failed to fetch record by ID")
		return errors.New("record not found")
	}
	logger.Log.WithFields(logrus.Fields{
		"id":    id,
		"model": model,
	}).Error("Record fetched Successfully!!")
	return nil
}

// GetAll records
func (g *GormDB) GetAll(model interface{}) error {
	if err := g.DB.Find(model).Error; err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch all records!!")
		return errors.New("failed to fetch records")
	}
	logrus.Info("All records fetched successfully!!")
	return nil
}

// GetByCondition records
func (g *GormDB) GetByCondition(model interface{}, condition string, args []interface{}, preloads []string, order string, limit, offset int) error {
	query := g.DB.Where(condition, args...)

	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	if order != "" {
		query = query.Order(order)
	}

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(model).Error; err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"model":     model,
			"condition": condition,
		}).Error("Failed to get record by Condition")
		return errors.New("failed to get record by condition")
	}
	logger.Log.WithFields(logrus.Fields{
		"model": model,
	}).Info("Record fetched successfully!!")
	return nil
}

// Update a record by ID
func (g *GormDB) Update(model interface{}, condition string, args []interface{}, updates interface{}) error {
	// Perform the update
	result := g.DB.Model(model).Clauses(clause.Returning{}).Where(condition, args...).Updates(updates).Update("updated_at", gorm.Expr("NOW()"))
	if result.Error != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":     result.Error,
			"model":     model,
			"condition": condition,
			"args":      args,
			"updates":   updates,
		}).Error("Failed to update records")
		return result.Error // Return the actual error
	}

	// Log the number of rows affected
	logger.Log.WithFields(logrus.Fields{
		"model":     model,
		"condition": condition,
		"args":      args,
		"updates":   updates,
		"rows":      result.RowsAffected,
	}).Info("Update operation completed")

	if result.RowsAffected == 0 {
		logger.Log.WithFields(logrus.Fields{
			"condition": condition,
			"args":      args,
		}).Warn("No records matched the update condition")
		return errors.New("no records matched the update condition")
	}

	return nil
}

// Delete a record by ID
func (g *GormDB) Delete(model interface{}, condition string, args []interface{}) error {
	if err := g.DB.Where(condition, args...).Delete(model).Error; err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"model": model,
		}).Error("Failed to delete record")
		return errors.New("failed to delete record")
	}
	logger.Log.WithFields(logrus.Fields{
		"model": model,
	}).Info("Record updated successfully!!")
	return nil
}

// Transaction Handler
func (g *GormDB) Transaction(fn func(tx *gorm.DB) error) error {
	tx := g.DB.Begin()
	if tx.Error != nil {
		logger.Log.WithError(tx.Error).Error("Failed to start transaction")
		return tx.Error
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		logger.Log.WithError(err).Error("Transaction failed, rolled back")
		return err
	}
	if err := tx.Commit().Error; err != nil {
		logger.Log.WithError(err).Error("Failed to commit transaction")
		return err
	}
	logger.Log.Info("Transaction committed successfully")
	return nil
}
