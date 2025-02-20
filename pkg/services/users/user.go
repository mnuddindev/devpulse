package users

import (
	"errors"

	"github.com/google/uuid"
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

func (us *UserSystem) BeforeCreate(client *grm.DB, user *models.User) (*models.User, error) {
	var memberRole models.Role
	if err := client.Where("name = ?", "member").First(&memberRole).Error; err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  "member",
		}).Error("User role can't be added")
		return nil, err
	}

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
	user.Roles = append(user.Roles, memberRole)
	user.Password = hashedPassword

	return user, nil
}

// CreateUser creates a new user in the system after validation and password hashing.
func (us *UserSystem) CreateUser(user *models.User) (*models.User, error) {
	user, err := us.BeforeCreate(us.crud.DB, user)
	if err != nil {
		return nil, err
	}
	user.NotificationsPreferences = []models.NotificationPrefrences{
		models.NotificationPrefrences{
			UserID:          user.ID,
			EmailOnLikes:    true,
			EmailOnComments: true,
			EmailOnMentions: true,
			EmailOnFollower: true,
			EmailOnBadge:    true,
			EmailOnUnread:   true,
			EmailOnNewPosts: true,
		},
	}

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
	if err := us.crud.GetByCondition(&user, "email = ?", []interface{}{email}, []string{}, "", 0, 0); err != nil {
		// log if failed to find the user with the provided credentials
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"email": email,
		}).Error("Failed to fetch user during login")
		return nil, errors.New("invalid credentials")
	}

	if user.ID.String() == "00000000-0000-0000-0000-000000000000" {
		logger.Log.WithFields(logrus.Fields{
			"error": "User not found",
		}).Warn("Unauthorized access attempt")
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

// UserProfile fetch everything about user from the db
func (us *UserSystem) UserProfile(id uuid.UUID) (models.User, error) {
	var user models.User
	err := us.crud.DB.Preload("Notifications").Select("id, username, first_name, last_name, bio, avatar_url, job_title, employer, location, github_url, website, skills, interests, is_active, theme_preference, created_at, updated_at").Where("id = ?", id).First(&user).Error
	if err != nil {
		if err == grm.ErrRecordNotFound {
			logger.Log.WithFields(logrus.Fields{
				"user_id": id,
			}).Warn("User not found")
		}
	}
	return user, err
}

// UserBy will fetch and filter out any data by given condition Like GetByLocation
func (us *UserSystem) UserBy(condition string, args ...interface{}) (*models.User, error) {
	// an empty instance of user model
	var user models.User

	// getting user details by given condition for instance ByLocation, BySkills, ByID
	if err := us.crud.GetByCondition(&user, condition, args, []string{"Badges", "Roles", "Followers", "Following", "Notifications", "NotificationsPreferences"}, "", 0, 0); err != nil {
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

func (us *UserSystem) ActiveUser(userid uuid.UUID) error {
	var user models.User
	updates := map[string]interface{}{
		"is_active":         true,
		"is_email_verified": true,
	}
	if err := us.crud.Update(&user, "id = ?", []interface{}{userid}, updates); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
			"id":    userid,
		}).Error("Failed to activate user")
		return err
	}
	return nil
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

// Users get all the users from the database
func (us *UserSystem) Users() ([]models.User, error) {
	// an empty instance of user model
	var users []models.User

	// check for users in db
	if err := us.crud.GetAll(&users, []string{"Badges", "Roles", "Followers", "Following", "Notifications", "NotificationsPreferences"}); err != nil {
		// log if failed to get data
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch all User")
		return nil, errors.New("failed to fetch all user")
	}

	//log if succed
	logger.Log.Info("All users fetched successfully!!")
	return users, nil
}

// Update Users updates user by ID
func (us *UserSystem) UpdateUser(condition string, userid uuid.UUID, updates interface{}) error {
	// an empty instance for user model
	var user models.User

	// delete user data using id
	if err := us.crud.Update(&user, condition, []interface{}{userid}, updates); err != nil {
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

// UpdateUserMany updates many2many field
func (us *UserSystem) UpdateUserMany(userid uuid.UUID, assoc string, userdata interface{}) {
	if err := us.crud.UpdateManyToMany(&models.User{ID: userid}, assoc, userdata); err != nil {
		logger.Log.Error(err)
	}
}

// DeleteUsers deletes user by ID
func (us *UserSystem) DeleteUser(userid uuid.UUID) error {
	// an empty instance for user model
	var user models.User

	// delete user data using id
	if err := us.crud.Delete(&user, "id = ?", []interface{}{userid}); err != nil {
		// log if failed
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"id":    userid,
		}).Error("Failed to delete user")
		return errors.New("failed to delete user")
	}
	logger.Log.WithFields(logrus.Fields{
		"id": userid,
	}).Info("User deleted Successfully!!")
	return nil
}
