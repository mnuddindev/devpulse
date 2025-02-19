package posts

import "github.com/mnuddindev/devpulse/pkg/services"

type PostsController struct {
	postSystem *services.UserSystem
}

func NewUserController(userSystem *services.UserSystem) *PostsController {
	return &PostsController{
		postSystem: userSystem,
	}
}
