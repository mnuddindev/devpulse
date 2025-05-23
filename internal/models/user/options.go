package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

func WithUsername(username string) UserOption {
	return func(u *User) { u.Username = username }
}

func WithEmail(email string) UserOption {
	return func(u *User) { u.Email = email }
}

func WithPassword(password string) UserOption {
	return func(u *User) { u.Password = password }
}

func WithPreviousPasswords(passwords string) UserOption {
	return func(u *User) { u.PreviousPasswords = passwords }
}

func WithPasswordChangedAt(changedAt time.Time) UserOption {
	return func(u *User) { u.LastPasswordChange = changedAt }
}

func WithOTP(otp string) UserOption {
	return func(u *User) { u.OTP = otp }
}

func WithIsActive(active bool) UserOption {
	return func(u *User) { u.IsActive = true; u.IsEmailVerified = true }
}

func WithEmailVerified(verified bool) UserOption {
	return func(u *User) { u.IsEmailVerified = verified }
}

func WithRole(roleName string) UserOption {
	return func(u *User) { u.Role = Role{Name: roleName} }
}

func WithRoleID(roleID uuid.UUID) UserOption {
	return func(u *User) { u.RoleID = roleID }
}

// Profile
func WithName(name string) UserOption {
	return func(u *User) { u.Profile.Name = name }
}

func WithBio(bio string) UserOption {
	return func(u *User) { u.Profile.Bio = bio }
}

func WithAvatarURL(url string) UserOption {
	return func(u *User) { u.Profile.AvatarURL = url }
}

func WithJobTitle(title string) UserOption {
	return func(u *User) { u.Profile.JobTitle = title }
}

func WithEmployer(employer string) UserOption {
	return func(u *User) { u.Profile.Employer = employer }
}

func WithLocation(location string) UserOption {
	return func(u *User) { u.Profile.Location = location }
}

func WithSocialLinks(links string) UserOption {
	return func(u *User) {
		if json, err := json.Marshal(links); err == nil {
			u.Profile.SocialLinks = string(json)
		}
	}
}

func WithCurrentLearning(learning string) UserOption {
	return func(u *User) { u.Profile.CurrentLearning = learning }
}

func WithAvailableFor(available string) UserOption {
	return func(u *User) { u.Profile.AvailableFor = available }
}

func WithCurrentlyHackingOn(hacking string) UserOption {
	return func(u *User) { u.Profile.CurrentlyHackingOn = hacking }
}

func WithPronouns(pronouns string) UserOption {
	return func(u *User) { u.Profile.Pronouns = pronouns }
}

func WithEducation(education string) UserOption {
	return func(u *User) { u.Profile.Education = education }
}

func WithSkills(skills []string) UserOption {
	return func(u *User) {
		if json, err := json.Marshal(skills); err == nil {
			u.Profile.Skills = string(json)
		}
	}
}

func WithInterests(interests []string) UserOption {
	return func(u *User) {
		if json, err := json.Marshal(interests); err == nil {
			u.Profile.Interests = string(json)
		}
	}
}

// Settings
func WithBrandColor(color string) UserOption {
	return func(u *User) { u.Settings.BrandColor = color }
}

func WithThemePreference(theme string) UserOption {
	return func(u *User) { u.Settings.ThemePreference = theme }
}

func WithBaseFont(font string) UserOption {
	return func(u *User) { u.Settings.BaseFont = font }
}

func WithSiteNavbar(navbar string) UserOption {
	return func(u *User) { u.Settings.SiteNavbar = navbar }
}

func WithContentEditor(editor string) UserOption {
	return func(u *User) { u.Settings.ContentEditor = editor }
}

func WithContentMode(mode int) UserOption {
	return func(u *User) { u.Settings.ContentMode = mode }
}

// Stats

func WithPostsCount(delta int) UserOption {
	return func(u *User) {
		u.Stats.PostsCount += delta
		if u.Stats.PostsCount < 0 {
			u.Stats.PostsCount = 0
		}
	}
}

func WithCommentsCount(delta int) UserOption {
	return func(u *User) {
		u.Stats.CommentsCount += delta
		if u.Stats.CommentsCount < 0 {
			u.Stats.CommentsCount = 0
		}
	}
}

func WithLikesCount(delta int) UserOption {
	return func(u *User) {
		u.Stats.LikesCount += delta
		if u.Stats.LikesCount < 0 {
			u.Stats.LikesCount = 0
		}
	}
}

func WithBookmarksCount(delta int) UserOption {
	return func(u *User) {
		u.Stats.BookmarksCount += delta
		if u.Stats.BookmarksCount < 0 {
			u.Stats.BookmarksCount = 0
		}
	}
}

func WithTagCount(delta int) UserOption {
	return func(u *User) {
		u.Stats.TagCount += delta
		if u.Stats.TagCount < 0 {
			u.Stats.TagCount = 0
		}
	}
}

func WithFollowersCount(delta int) UserOption {
	return func(u *User) {
		u.Stats.FollowersCount += delta
		if u.Stats.FollowersCount < 0 {
			u.Stats.FollowersCount = 0
		}
	}
}

func WithFollowingCount(delta int) UserOption {
	return func(u *User) {
		u.Stats.FollowingCount += delta
		if u.Stats.FollowingCount < 0 {
			u.Stats.FollowingCount = 0
		}
	}
}

func WithReactionsCount(delta int) UserOption {
	return func(u *User) {
		u.Stats.ReactionsCount += delta
		if u.Stats.ReactionsCount < 0 {
			u.Stats.ReactionsCount = 0
		}
	}
}

func WithLastSeen(lastSeen time.Time) UserOption {
	return func(u *User) { u.Stats.LastSeen = lastSeen }
}

// stats
func WithEmailOnLikes(ok bool) UserOption {
	return func(u *User) { u.NotificationPreferences.EmailOnLikes = ok }
}

func WithEmailOnComments(ok bool) UserOption {
	return func(u *User) { u.NotificationPreferences.EmailOnComments = ok }
}

func WithEmailOnMentions(ok bool) UserOption {
	return func(u *User) { u.NotificationPreferences.EmailOnMentions = ok }
}

func WithEmailOnFollowers(ok bool) UserOption {
	return func(u *User) { u.NotificationPreferences.EmailOnFollowers = ok }
}

func WithEmailOnBadge(ok bool) UserOption {
	return func(u *User) { u.NotificationPreferences.EmailOnBadge = ok }
}

func WithEmailOnUnread(ok bool) UserOption {
	return func(u *User) { u.NotificationPreferences.EmailOnUnread = ok }
}

func WithEmailOnNewPosts(ok bool) UserOption {
	return func(u *User) { u.NotificationPreferences.EmailOnNewPosts = ok }
}
