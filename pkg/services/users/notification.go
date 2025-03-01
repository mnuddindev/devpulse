package users

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CreateUser creates a new user in the system after validation and password hashing.
func (us *UserSystem) CreateNotification(userid uuid.UUID, notification models.Notification) error {
	// Attempt to create the user in the database. If creation fails, log the error and return an error.
	if err := us.Crud.Create(notification); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userid": userid,
		}).Error("failed to register user")
		return errors.New("failed to input default notification prefrences")
	}

	// Log the successful creation of the user.
	logger.Log.WithFields(logrus.Fields{
		"userid": userid,
	}).Info("Notification Preferences added successfully!!")

	// Return the created user and no error.
	return nil
}

// UserBy will fetch and filter out any data by given condition Like GetByLocation
func (us *UserSystem) NotificationBy(condition string, args ...interface{}) (*models.Notification, error) {
	// an empty instance of user model
	var noti models.Notification

	// getting user details by given condition for instance ByLocation, BySkills, ByID
	if err := us.Crud.GetByCondition(&noti, condition, args, []string{}, "", 0, 0); err != nil {
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
	if err := us.Crud.GetByCondition(&noti, condition, args, []string{}, "", 0, 0); err != nil {
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
	if err := uc.Crud.GetAll(&noti, []string{}); err != nil {
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

func (uc *UserSystem) NotificationPref() ([]models.NotificationPrefrences, error) {
	// an empty instance of user model
	var noti []models.NotificationPrefrences

	// check for users in db
	if err := uc.Crud.GetAll(&noti, []string{}); err != nil {
		// log if failed to get data
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch all Notification Preferences")
		return nil, errors.New("failed to fetch all Notification Preferences")
	}

	//log if succed
	logger.Log.Info("All users fetched successfully!!")
	return noti, nil
}

// Update Users updates user by ID
func (us *UserSystem) UpdateNotification(condition string, notid uuid.UUID, updates interface{}) error {
	// an empty instance for user model
	var notification models.Notification

	// delete user data using id
	if err := us.Crud.Update(&notification, "id = ?", []interface{}{notid}, updates); err != nil {
		// log if failed
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"id":    notid,
		}).Error("Failed to update User's notificatio")
		return errors.New("failed to update User's notificatio")
	}
	logger.Log.WithFields(logrus.Fields{
		"id": notid,
	}).Info("Notification updated Successfully!!")
	return nil
}

func (us *UserSystem) UpdateNotificationPref(condition string, userid uuid.UUID, updates interface{}) (*models.NotificationPrefrences, error) {
	var prefs models.NotificationPrefrences
	err := us.Crud.DB.Model(&prefs).Where(condition, userid).Updates(updates).Select("*").First(&prefs).Error
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"id":    userid,
		}).Error("Failed to update notification preferences")
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("notification preferences not found")
		}
		return nil, errors.New("failed to update notification preferences")
	}
	return &prefs, nil
}
