package v1

import "github.com/gofiber/fiber/v2"

func Register(c *fiber.Ctx) error {
	type UserInput struct {
		AvatarURL       string `json:"avatar_url"`
		Name            string `json:"name"`
		Username        string `json:"username"`
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
	}
}
