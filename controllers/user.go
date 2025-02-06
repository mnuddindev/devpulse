package controllers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/auth"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/mnuddindev/devpulse/pkg/services"
	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/sirupsen/logrus"
)

type UserController struct {
	userSystem *services.UserSystem
}

func NewUserController(userSystem *services.UserSystem) *UserController {
	return &UserController{
		userSystem: userSystem,
	}
}

// Registration handles user registration
func (uc *UserController) Registration(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Invalid request payload")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": err.Error(),
			"status": fiber.StatusBadRequest,
		})
	}
	validator := utils.NewValidator()
	if err := validator.Validate(user); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  user,
		}).Error("User validation failed while registering")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}
	otp, err := utils.GenerateOTP()
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"field": "OTP Generation",
		}).Error("OTP Generation failed")
	}
	user.OTP = otp
	newUser, err := uc.userSystem.CreateUser(&user)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to register user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  err.Error(),
			"status": fiber.StatusInternalServerError,
		})
	}

	utils.SendActivationEmail(otp, newUser.Email, newUser.Username)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user": fiber.Map{
			"userid":   newUser.ID,
			"username": newUser.Username,
		},
		"message": "User registered successfully!!",
	})
}

// ActiveUser verifies user by otp
func (uc *UserController) ActiveUser(c *fiber.Ctx) error {
	// parse request body
	type Body struct {
		Otp int64 `json:"otp"`
	}
	var body Body
	if err := c.BodyParser(&body); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}
	fmt.Println(body)

	// validate user id
	userID, err := uuid.Parse(c.Params("userid"))
	if err != nil || userID == uuid.Nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Invalid user ID")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "User not found",
			"status": fiber.StatusNotFound,
		})
	}

	// check userID is not valid
	if userID.String() == "00000000-0000-0000-0000-000000000000" {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("User not found")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errors": "user not found",
			"status": fiber.StatusNotFound,
		})
	}

	// Fetch user ID
	user, err := uc.userSystem.UserBy("id = ?", userID)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Failed to fetch user by ID")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "User not found",
			"status": fiber.StatusNotFound,
		})
	}

	// validate OTP
	if body.Otp != user.OTP {
		logger.Log.WithFields(logrus.Fields{
			"userID": userID,
		}).Error("OTP mismatch")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "OTP not matched",
			"status": fiber.StatusBadRequest,
		})
	}

	// check if user already verified
	if user.IsActive {
		logger.Log.WithFields(logrus.Fields{
			"userID": userID,
		}).Error("OTP expired or already verified")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "OTP expired or already verified",
			"status": fiber.StatusBadRequest,
		})
	}

	// Activate user if not activated
	if err := uc.userSystem.ActiveUser(userID); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error":  err,
			"userID": userID,
		}).Error("Failed to activate user")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to activate account",
			"status": fiber.StatusInternalServerError,
		})
	}

	// return success response
	logger.Log.WithFields(logrus.Fields{
		"userID": userID,
	}).Info("User activated successfully")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  fiber.StatusOK,
		"message": "Account activated successfully",
		"data": fiber.Map{
			"user_id":       user.ID,
			"name":          user.FirstName + " " + user.LastName,
			"email":         user.Email,
			"profile_photo": user.AvatarUrl,
			"message":       "Your account has been activated. Please log in now!",
		},
	})
}

func (uc *UserController) Login(c *fiber.Ctx) error {
	type Login struct {
		Email    string `json:"email" validate:"required,email,min=5"`
		Password string `json:"password" validate:"required,min=6"`
	}
	// parse request body
	var login Login
	if err := c.BodyParser(&login); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid request body",
			"status": fiber.StatusBadRequest,
		})
	}

	// validate email password
	validator := utils.NewValidator()
	if err := validator.Validate(login); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"user":  login,
		}).Error("User validation failed while registering")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"errors": err,
			"status": fiber.StatusUnprocessableEntity,
		})
	}

	// fetch user by email
	user, err := uc.userSystem.UserBy("email = ?", login.Email)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"email": login.Email,
		}).Error("Failed to fetch user by email")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "User not found",
			"status": fiber.StatusNotFound,
		})
	}

	// compare user password
	if err := utils.ComparePasswords(user.Password, login.Password); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"email": login.Email,
		}).Error("Password mismatch")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Email or password not matched",
			"status": fiber.StatusUnauthorized,
		})
	}

	// check if user is activated
	if !user.IsActive {
		logger.Log.WithFields(logrus.Fields{
			"email": login.Email,
		}).Error("User not verified")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Verify your account first",
			"status": fiber.StatusUnauthorized,
		})
	}

	// Generate JWT tokens
	atoken, rtoken, err := auth.GenerateJWT(user.ID, user.Email)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to generate JWT tokens")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":  "Token generation failed",
			"status": fiber.StatusUnprocessableEntity,
		})
	}
	at := fiber.Cookie{
		Name:     "access_token",
		Value:    atoken,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
	}
	rt := fiber.Cookie{
		Name:     "refresh_token",
		Value:    rtoken,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: true,
	}
	c.Cookie(&at)
	c.Cookie(&rt)
	logger.Log.WithFields(logrus.Fields{
		"email": login.Email,
	}).Info("User logged in successfully")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login successful",
		"status":  fiber.StatusOK,
		"data": fiber.Map{
			"user_id":       user.ID,
			"name":          user.FirstName + " " + user.LastName,
			"email":         user.Email,
			"profile_photo": user.AvatarUrl,
		},
	})
}
