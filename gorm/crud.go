package gorm

import (
	"errors"

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
	Update(model interface{}, id int) error
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
		logrus.WithFields(logrus.Fields{
			"error": err,
			"model": model,
		}).Error("Error while creating a new record")
		return errors.New("error while creating a new record")
	}
	logrus.WithFields(logrus.Fields{
		"model": model,
	}).Info("Record created succesfully!!")
	return nil
}

// GetByID a record by ID
func (g *GormDB) GetByID(model interface{}, id int) error {
	if err := g.DB.First(model, id).Error; err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
			"id":    id,
		}).Error("Failed to fetch record by ID")
		return errors.New("record not found")
	}
	logrus.WithFields(logrus.Fields{
		"id":    id,
		"model": model,
	}).Error("Record fetched Successfully!!")
	return nil
}

// GetAll records
func (g *GormDB) GetAll(model interface{}) error {
	if err := g.DB.Find(model).Error; err != nil {
		logrus.WithFields(logrus.Fields{
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
		logrus.WithFields(logrus.Fields{
			"error":     err,
			"model":     model,
			"condition": condition,
		}).Error("Failed to get record by Condition")
		return errors.New("failed to get record by condition")
	}
	logrus.WithFields(logrus.Fields{
		"model": model,
	}).Info("Record fetched successfully!!")
	return nil
}

// Update a record by ID
func (g *GormDB) Update(model interface{}, id int) error {
	if err := g.DB.Save(model).Error; err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
			"model": model,
		}).Error("Failed to update record")
		return errors.New("failed to update record")
	}
	logrus.WithFields(logrus.Fields{
		"model": model,
	}).Info("Record updated successfully!!")
	return nil
}

// Delete a record by ID
func (g *GormDB) Delete(model interface{}, id int) error {
	if err := g.DB.Delete(model, id).Error; err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
			"model": model,
		}).Error("Failed to delete record")
		return errors.New("failed to delete record")
	}
	logrus.WithFields(logrus.Fields{
		"model": model,
	}).Info("Record updated successfully!!")
	return nil
}
