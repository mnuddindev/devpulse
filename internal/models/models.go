package models

import user "github.com/mnuddindev/devpulse/internal/models/user"

func RegisterModels() []interface{} {
	return []interface{}{
		&user.User{},
		&user.Role{},
		&user.Permission{},
		&user.Notification{},
		&user.NotificationPreferences{},
		&user.Badge{},
	}
}

type (
	User                    = user.User
	Role                    = user.Role
	Permission              = user.Permission
	Notification            = user.Notification
	NotificationPreferences = user.Notification
	Badge                   = user.Badge
)

var (
	SeedRoles = user.SeedRoles
)
