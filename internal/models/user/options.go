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

func WithOTP(otp int64) UserOption {
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

func WithSocialLinks(github, website, twitter string) UserOption {
	return func(u *User) {
		links := map[string]string{"github": github, "website": website, "twitter": twitter}
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
func WithPostsCount(count int) UserOption {
	return func(u *User) { u.Stats.PostsCount = count }
}

func WithCommentsCount(count int) UserOption {
	return func(u *User) { u.Stats.CommentsCount = count }
}

func WithLikesCount(count int) UserOption {
	return func(u *User) { u.Stats.LikesCount = count }
}

func WithBookmarksCount(count int) UserOption {
	return func(u *User) { u.Stats.BookmarksCount = count }
}

func WithLastSeen(lastSeen time.Time) UserOption {
	return func(u *User) { u.Stats.LastSeen = lastSeen }
}
