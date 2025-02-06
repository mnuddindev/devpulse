package controllers

import (
	"github.com/gofiber/fiber/v2"
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
		"message": "User registered successfully!!",
	})
}
