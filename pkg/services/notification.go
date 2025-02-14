package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
)

// UserBy will fetch and filter out any data by given condition Like GetByLocation
func (us *UserSystem) NotificationBy(condition string, args ...interface{}) (*models.Notification, error) {
	// an empty instance of user model
	var noti models.Notification

	// getting user details by given condition for instance ByLocation, BySkills, ByID
	if err := us.crud.GetByCondition(&noti, condition, args, []string{}, "", 0, 0); err != nil {
		// log if failed to fetch by condition
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to fetch notification by Condition")
		return nil, errors.New("notification not found!!")
	}

	// log if successfully fetched the use data by condition
	logger.Log.WithFields(logrus.Fields{
		"notification": noti,
	}).Info("notification Fetched Successfully!!")

	// return the user data and error
	return &noti, nil
}

// UserBy will fetch and filter out any data by given condition Like GetByLocation
func (us *UserSystem) NotificationPreBy(condition string, args ...interface{}) (*models.NotificationPrefrences, error) {
	// an empty instance of user model
	var noti models.NotificationPrefrences

	// getting user details by given condition for instance ByLocation, BySkills, ByID
	if err := us.crud.GetByCondition(&noti, condition, args, []string{}, "", 0, 0); err != nil {
		// log if failed to fetch by condition
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to fetch notificationpre by Condition")
		return nil, errors.New("notificationpre not found!!")
	}

	// log if successfully fetched the use data by condition
	logger.Log.WithFields(logrus.Fields{
		"notificationpre": noti,
	}).Info("notificationpre Fetched Successfully!!")

	// return the user data and error
	return &noti, nil
}

// Users get all the users from the database
func (uc *UserSystem) Notification() ([]models.Notification, error) {
	// an empty instance of user model
	var noti []models.Notification

	// check for users in db
	if err := uc.crud.GetAll(&noti); err != nil {
		// log if failed to get data
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch all User")
		return nil, errors.New("failed to fetch all user")
	}

	//log if succed
	logger.Log.Info("All users fetched successfully!!")
	return noti, nil
}

// Update Users updates user by ID
func (us *UserSystem) UpdateNotification(condition string, userid uuid.UUID, updates interface{}) error {
	// an empty instance for user model
	var user models.User

	// delete user data using id
	if err := us.crud.Update(&user, "id = ?", []interface{}{userid}, updates); err != nil {
		// log if failed
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"id":    userid,
		}).Error("Failed to delete user")
		return errors.New("failed to delete user")
	}
	logger.Log.WithFields(logrus.Fields{
		"id": userid,
	}).Info("User updated Successfully!!")
	return nil
}

// Update Users Notification preferences updates by ID
func (us *UserSystem) UpdateNotificationPref(condition string, notifid uuid.UUID, updates interface{}) error {
	// an empty instance for user model
	var notificationpre models.NotificationPrefrences

	// delete user data using id
	if err := us.crud.Update(&notificationpre, "id = ?", []interface{}{notifid}, updates); err != nil {
		// log if failed
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"id":    notifid,
		}).Error("Failed to update users notification preferences")
		return errors.New("failed to update users notification preferences")
	}
	logger.Log.WithFields(logrus.Fields{
		"id": notifid,
	}).Info("Users Notification Preferences updated Successfully!!")
	return nil
}
