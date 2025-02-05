package services

import (
	"errors"

	"github.com/mnuddindev/devpulse/gorm"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/sirupsen/logrus"
	grm "gorm.io/gorm"
)

// UserSystem struct that holds a reference to the CRUD operations using Gorm.
type UserSystem struct {
	crud *gorm.GormDB
}

// NewUserSystem initializes a new UserSystem with a given database connection.
func NewUserSystem(db *grm.DB) *UserSystem {
	return &UserSystem{
		crud: gorm.NewGormDB(db),
	}
}

// CreateUser creates a new user in the system after validation and password hashing.
func (us *UserSystem) CreateUser(user *models.User) (*models.User, error) {
	// Initialize a new validator instance.
	validate := utils.NewValidator()

	// Validate the user data. If validation fails, log the error and return an error.
	if err := validate.Validate(user); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  user,
		}).Error("Validation failed during creating user")
		return &models.User{}, errors.New("invalid user data")
	}

	// Hash the user's password. If hashing fails, log the error and return an error.
	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to hash password while creating user")
		return &models.User{}, errors.New("failed to process user data")
	}

	// Set the hashed password back to the user.
	user.Password = hashedPassword

	// Attempt to create the user in the database. If creation fails, log the error and return an error.
	if err := us.crud.Create(user); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  user,
		}).Error("failed to register user")
		return &models.User{}, errors.New("failed to register user")
	}

	// Log the successful creation of the user.
	logger.Log.WithFields(logrus.Fields{
		"user": user,
	}).Info("User registered successfully!!")

	// Return the created user and no error.
	return user, nil
}

func (us *UserSystem) LoginUser(email, password string) (*models.User, error) {
	// an empty instance of user model
	var user models.User

	// trying to get user by email to match credentials
	if err := us.crud.GetByCondition(&user, "email = ?", email); err != nil {
		// log if failed to find the user with the provided credentials
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"email": email,
		}).Error("Failed to fetch user during login")
		return nil, errors.New("invalid credentials")
	}

	// Check password if the given password is correct
	if err := utils.ComparePasswords(password, user.Password); err != nil {
		// log if the user given password and the user password in db does not match
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Invalid password during login")
		return nil, errors.New("invalid credentials")
	}

	// log if logged in successfully
	logger.Log.WithFields(logrus.Fields{
		"user": user,
	}).Info("User Logged in Successfully!!")
	return &user, nil
}

// UserBy will fetch and filter out any data by given condition Like GetByLocation
func (us *UserSystem) UserBy(condition string, args ...interface{}) (*models.User, error) {
	// an empty instance of user model
	var user models.User

	// getting user details by given condition for instance ByLocation, BySkills, ByID
	if err := us.crud.GetByCondition(&user, condition, args...); err != nil {
		// log if failed to fetch by condition
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to fetch user by Condition")
		return nil, errors.New("user not found!!")
	}

	// log if successfully fetched the use data by condition
	logger.Log.WithFields(logrus.Fields{
		"user": user,
	}).Info("User Fetched Successfully!!")

	// return the user data and error
	return &user, nil
}

// UserActiveByID checks if the user is activated by fetching data using it's ID
func (us *UserSystem) UserActiveByID(userid string) (bool, error) {
	// an empty user model
	var user models.User

	// check and store data if the user is activated or not
	if err := us.crud.GetByID(&user, userid); err != nil {
		// log if failed to fetch user by ID
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"id":    userid,
		}).Error("Failed to fetch user by ID")
		// return false if user not found
		return false, errors.New("user not found")
	}

	// log if user found
	logger.Log.WithFields(logrus.Fields{
		"user": user,
	}).Info("User active status checked successfully!!")
	return true, nil
}

// GetOTP generates ONE TIME PASSWORD and assigns an otp the user
func (us *UserSystem) GetOTP(email string) (int, error) {
	// empty instance of model user
	var user models.User

	// check if user available
	if err := us.crud.GetByCondition(&user, "email = ?", email); err != nil {
		// log if failed to gather user
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"email": email,
		}).Error("Failed to fetch usr for OTP generation")
		return 0, errors.New("user not found")
	}

	// generating otp
	otp, err := utils.GenerateOTP()
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"field": "OTP Generation",
		}).Error("OTP Generation failed")
		return 0, errors.New("otp generation failed")
	}

	// set OTP to user
	user.OTP = int(otp)

	if err := us.crud.Update(&user, user.ID); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  user,
		}).Error("Failed to update user with OTP")
		return 0, errors.New("failed to generate OTP")
	}

	// log if succeded
	logger.Log.WithFields(logrus.Fields{
		"user": user,
	}).Info("OTP generated and assigned successfully")
	return user.OTP, nil
}
