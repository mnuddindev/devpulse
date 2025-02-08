package gorm

import (
	"errors"
	"strings"

	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
func (g *GormDB) GetByCondition(model interface{}, condition string, args ...interface{}) error {
	if err := g.DB.Where(condition, args...).Find(model).Error; err != nil {
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
func (g *GormDB) Update(model interface{}, condition string, args interface{}, updates interface{}) error {
	// Find the record by ID or condition
	result := g.DB.Model(model).Where(condition, args).Updates(updates)
	if result.Error != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":     result.Error,
			"model":     model,
			"condition": condition,
			"args":      args,
			"updates":   updates,
		}).Error("Failed to update records")
		return errors.New("failed to update records")
	}

	// Check if any records were updated
	if result.RowsAffected == 0 {
		logger.Log.WithFields(logrus.Fields{
			"condition": condition,
			"args":      args,
		}).Warn("No records matched the update condition")
		return errors.New("no records matched the update condition")
	}

	logger.Log.WithFields(logrus.Fields{
		"model":     model,
		"condition": condition,
		"args":      args,
		"updates":   updates,
		"rows":      result.RowsAffected,
	}).Info("Records updated successfully")
	return nil
}

// Delete a record by ID
func (g *GormDB) Delete(model interface{}, id interface{}) error {
	if err := g.DB.Delete(model, id).Error; err != nil {
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

// updateManyToMany updates many2many connections
func (g *GormDB) updateManyToMany(userField interface{}, names []string, field string, model interface{}) {
	if names == nil {
		return
	}

	// Slice to hold th update records
	var newItems []interface{}

	// Loop through the names and find or create records
	for _, name := range names {
		item := model
		g.DB.FirstOrCreate(&item, field+" = ?", strings.ToLower(name))
		newItems = append(newItems, item)
	}

	switch fieldPtr := userField.(type) {
	case *[]models.Skill:
		*fieldPtr = newItemsToType[models.Skill](newItems)
	case *[]models.Interest:
		*fieldPtr = newItemsToType[models.Interest](newItems)
	case *[]models.Badge:
		*fieldPtr = newItemsToType[models.Badge](newItems)
	case *[]models.Role:
		*fieldPtr = newItemsToType[models.Role](newItems)
	default:
		logrus.Warnf("updateManyToMany: Unhandled type %T", userField)
	}
}

// newItemsToType converts []interface{} to the correct type slice
func newItemsToType[T any](items []interface{}) []T {
	var result []T
	for _, item := range items {
		if typedItem, ok := item.(T); ok {
			result = append(result, typedItem)
		}
	}
	return result
}
