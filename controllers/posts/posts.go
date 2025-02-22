package postscontroller

import (
	"github.com/gofiber/fiber/v2"
	postservices "github.com/mnuddindev/devpulse/pkg/services/posts"
)

type PostController struct {
	postSystem *postservices.PostSystem
}

func NewPostController(postSystem *postservices.PostSystem) *PostController {
	return &PostController{
		postSystem: postSystem,
	}
}

// CreatePost creates a new post
func (pc *PostController) CreatePost(c *fiber.Ctx) error {

}
