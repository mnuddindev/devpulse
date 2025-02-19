package users

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/sirupsen/logrus"
)

func (uc *UserController) FollowUser(c *fiber.Ctx) error {
	following, err := uuid.Parse(c.Params("userid"))
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": "Invalid user id",
		}).Error("Invalid user id")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"status":   fiber.StatusUnprocessableEntity,
			"messagee": "Inavlid user id, failed to find user",
		})
	}
	// Get user ID from context
	follower, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		logger.Log.WithFields(logrus.Fields{
			"error": "User ID missing or invalid type in context",
		}).Warn("Unauthorized access attempt")
		// Return unauthorized status
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	fname, fname2, err := uc.userSystem.Follow(follower, following)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn(err.Error())
		// Return unauthorized status
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  err.Error(),
			"status": fiber.StatusNotFound,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  fiber.StatusOK,
		"message": fname + " followed " + fname2,
	})
}

func (uc *UserController) UnfollowUser(c *fiber.Ctx) error {
	following, err := uuid.Parse(c.Params("userid"))
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": "Invalid user id",
		}).Error("Invalid user id")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"status":   fiber.StatusUnprocessableEntity,
			"messagee": "Inavlid user id, failed to find user",
		})
	}
	// Get user ID from context
	follower, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		logger.Log.WithFields(logrus.Fields{
			"error": "User ID missing or invalid type in context",
		}).Warn("Unauthorized access attempt")
		// Return unauthorized status
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":  "Unauthorized",
			"status": fiber.StatusUnauthorized,
		})
	}

	fname, fname2, err := uc.userSystem.Unfollow(follower, following)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn(err.Error())
		// Return unauthorized status
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  err.Error(),
			"status": fiber.StatusNotFound,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  fiber.StatusOK,
		"message": fname + " unfollowed " + fname2,
	})
}

func (uc *UserController) GetAllFollowers(c *fiber.Ctx) error {
	userid, err := uuid.Parse(c.Params("userid"))
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": "Invalid user id",
		}).Error("Invalid user id")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"status":   fiber.StatusUnprocessableEntity,
			"messagee": "Inavlid user id, failed to find user",
		})
	}

	followers, err := uc.userSystem.GetFollowers(userid)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn(err.Error())
		// Return unauthorized status
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  err.Error(),
			"status": fiber.StatusNotFound,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":    fiber.StatusOK,
		"followers": followers,
	})
}

func (uc *UserController) GetAllFollowing(c *fiber.Ctx) error {
	userid, err := uuid.Parse(c.Params("userid"))
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": "Invalid user id",
		}).Error("Invalid user id")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"status":   fiber.StatusUnprocessableEntity,
			"messagee": "Inavlid user id, failed to find user",
		})
	}

	following, err := uc.userSystem.GetFollowing(userid)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Warn(err.Error())
		// Return unauthorized status
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  err.Error(),
			"status": fiber.StatusNotFound,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":    fiber.StatusOK,
		"following": following,
	})
}
