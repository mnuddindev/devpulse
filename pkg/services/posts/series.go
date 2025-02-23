package postservices

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
)

// CreateSeries creates a new series in the database.
func (ps *PostSystem) CreateSeries(series *models.Series) (*models.Series, error) {
	err := ps.Crud.Create(series)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error creating series")
		return nil, errors.New("error creating series")
	}

	return series, nil
}

// GetSeriesBy retrieves a series by a given condition.
func (ps *PostSystem) GetSeriesBy(condition string, args ...interface{}) (*models.Series, error) {
	// an empty instance of series model
	var series models.Series

	// getting series details by given condition
	if err := ps.Crud.GetByCondition(&series, condition, args, []string{"Author", "Posts", "Analytics"}, "", 0, 0); err != nil {
		// log if failed to fetch by condition
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to fetch series by Condition")
		return nil, errors.New("series not found!!")
	}

	// log if successfully fetched the use data by condition
	logger.Log.WithFields(logrus.Fields{
		"series": series,
	}).Info("User Fetched Successfully!!")

	// return the comment data and error
	return &series, nil
}

// UpdateSeries updates a series in the database.
func (ps *PostSystem) UpdateSeries(series *models.Series) (*models.Series, error) {
	err := ps.Crud.Update(&series, "id = ?", []interface{}{series.ID}, series)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error updating series")
		return nil, errors.New("error updating series")
	}

	// return all updated field with preload using getconditionby
	if err := ps.Crud.GetByCondition(series, "id = ?", []interface{}{series.ID}, []string{"Author", "Posts", "Analytics"}, "", 0, 0); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch series by Condition")
		return nil, errors.New("series not found")
	}

	return series, nil
}

// DeleteSeries deletes a series from the database.
func (ps *PostSystem) DeleteSeries(id string) error {
	series := &models.Series{}
	err := ps.Crud.Delete(series, "id = ?", []interface{}{id})
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error deleting series")
		return errors.New("error deleting series")
	}
	return nil
}

// Series retrieves all series from the database.
func (ps *PostSystem) Series() ([]models.Series, error) {
	series := []models.Series{}
	err := ps.Crud.GetAll(&series, []string{"Author", "Posts", "Analytics"})
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error fetching all series")
		return nil, errors.New("error fetching all series")
	}
	return series, nil
}

// UpdateSeriesMany updates a many-to-many field in the database.
func (ps *PostSystem) UpdateSeriesMany(seriesID uuid.UUID, assoc string, data interface{}) error {
	err := ps.Crud.UpdateManyToMany(&models.Series{ID: seriesID}, assoc, data)
	if err != nil {
		logger.Log.Error(err)
		return errors.New("error updating many to many")
	}
	return nil
}
