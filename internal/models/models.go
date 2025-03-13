package models

import user "github.com/mnuddindev/devpulse/internal/models/user"

func RegisterModels() []interface{} {
	return []interface{}{
		&user.User{},
		&user.Role{},
		&user.Permission{},
		//&user.Badge{},
		&user.Notification{},
		&user.NotificationPreferences{},
	}
}

type (
	User                    = user.User
	Role                    = user.Role
	Permission              = user.Permission
	Badge                   = user.Badge
	Notification            = user.Notification
	NotificationPreferences = user.NotificationPreferences
)

var (
	NewUser                    = user.NewUser
	NewRole                    = user.NewRole
	NewPermission              = user.NewPermission
	NewBadge                   = user.NewBadge
	NewNotification            = user.NewNotification
	NewNotificationPreferences = user.NewNotificationPreferences
	SeedRoles                  = user.SeedRoles
)
