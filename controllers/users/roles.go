package users

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func (uc *UserController) AddRoleToUser(c *fiber.Ctx) error {
	type Request struct {
		UserID uuid.UUID `json:"user_id" validate:"required"`
		RoleID uuid.UUID `json:"role_id" validate:"required"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Validate input
	if req.UserID == uuid.Nil || req.RoleID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User ID and Role ID are required"})
	}

	user, err := uc.userSystem.UserBy("id = ?", req.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	role, err := uc.userSystem.RoleBy("id = ?", req.RoleID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
	}

	// Add role to user
	if err := uc.userSystem.Crud.AddManyToMany(&user, "Roles", &role); err != nil {
		logrus.WithError(err).Error("Failed to assign role")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to assign role"})
	}

	logrus.WithFields(logrus.Fields{
		"user_id": req.UserID,
		"role":    role.Name,
	}).Info("Role assigned to user")
	return c.JSON(fiber.Map{"message": "Role assigned successfully"})
}
