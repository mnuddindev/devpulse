package models

import (
	posts "github.com/mnuddindev/devpulse/internal/models/posts"
	user "github.com/mnuddindev/devpulse/internal/models/user"
)

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
	UpdateUserRequest       = user.UpdateUserRequest
	Role                    = user.Role
	Permission              = user.Permission
	Badge                   = user.Badge
	Notification            = user.Notification
	NotificationPreferences = user.NotificationPreferences
	UserOption              = user.UserOption

	Posts            = posts.Posts
	PostAnalytics    = posts.PostAnalytics
	Series           = posts.Series
	SeriesPost       = posts.SeriesPost
	SeriesAnalytics  = posts.SeriesAnalytics
	Bookmark         = posts.Bookmark
	Collection       = posts.Collection
	Tag              = posts.Tag
	TagFollower      = posts.TagFollower
	TagAnalytics     = posts.TagAnalytics
	TagModerator     = posts.TagModerator
	Reaction         = posts.Reaction
	ReadingListEntry = posts.ReadingListEntry
	Comment          = posts.Comment
	CommentFlag      = posts.CommentFlag
	CommentMention   = posts.CommentMention
)

var (
	NewUser         = user.NewUser
	GetUserBy       = user.GetUserBy
	GetUsers        = user.GetUsers
	UpdateUser      = user.UpdateUser
	UpdateUserStats = user.UpdateUserStats
	DeleteUser      = user.DeleteUser

	WithUsername           = user.WithUsername
	WithEmail              = user.WithEmail
	WithPassword           = user.WithPassword
	WithPreviousPasswords  = user.WithPreviousPasswords
	WithPasswordChangedAt  = user.WithPasswordChangedAt
	WithOTP                = user.WithOTP
	WithIsActive           = user.WithIsActive
	WithEmailVerified      = user.WithEmailVerified
	WithRole               = user.WithRole
	WithRoleID             = user.WithRoleID
	WithName               = user.WithName
	WithBio                = user.WithBio
	WithAvatarURL          = user.WithAvatarURL
	WithJobTitle           = user.WithJobTitle
	WithEmployer           = user.WithEmployer
	WithLocation           = user.WithLocation
	WithSocialLinks        = user.WithSocialLinks
	WithCurrentLearning    = user.WithCurrentLearning
	WithAvailableFor       = user.WithAvailableFor
	WithCurrentlyHackingOn = user.WithCurrentlyHackingOn
	WithPronouns           = user.WithPronouns
	WithEducation          = user.WithEducation
	WithSkills             = user.WithSkills
	WithInterests          = user.WithInterests
	WithBrandColor         = user.WithBrandColor
	WithThemePreference    = user.WithThemePreference
	WithBaseFont           = user.WithThemePreference
	WithSiteNavbar         = user.WithThemePreference
	WithContentEditor      = user.WithContentEditor
	WithContentMode        = user.WithContentMode
	WithPostsCount         = user.WithPostsCount
	WithCommentsCount      = user.WithCommentsCount
	WithLikesCount         = user.WithLikesCount
	WithBookmarksCount     = user.WithBookmarksCount
	WithLastSeen           = user.WithLastSeen
	WithEmailOnLikes       = user.WithEmailOnLikes
	WithEmailOnComments    = user.WithEmailOnComments
	WithEmailOnMentions    = user.WithEmailOnMentions
	WithEmailOnFollowers   = user.WithEmailOnFollowers
	WithEmailOnBadge       = user.WithEmailOnBadge
	WithEmailOnUnread      = user.WithEmailOnUnread
	WithEmailOnNewPosts    = user.WithEmailOnNewPosts

	NewRole       = user.NewRole
	GetRoleBy     = user.GetRoleBy
	GetRoles      = user.GetRoles
	UpdateRole    = user.UpdateRole
	DeleteRole    = user.DeleteRole
	NewPermission = user.NewPermission
	NewBadge      = user.NewBadge
	SeedRoles     = user.SeedRoles

	NewNotification    = user.NewNotification
	GetNotification    = user.GetNotification
	GetNotifications   = user.GetNotifications
	UpdateNotification = user.UpdateNotification
	DeleteNotification = user.DeleteNotification

	NewNotificationPreferences    = user.NewNotificationPreferences
	UpdateNotificationPreferences = user.UpdateNotificationPreferences
)
