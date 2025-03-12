package models

import "encoding/json"

func WithName(name string) UserOption {
	return func(u *User) { u.Profile.Name = name }
}

func WithBio(bio string) UserOption {
	return func(u *User) { u.Profile.Bio = bio }
}

func WithAvatarURL(url string) UserOption {
	return func(u *User) { u.Profile.AvatarURL = url }
}

func WithSocialLinks(github, website, twitter string) UserOption {
	return func(u *User) {
		links := map[string]string{"github": github, "website": website, "twitter": twitter}
		if json, err := json.Marshal(links); err == nil {
			u.Profile.SocialLinks = string(json)
		}
	}
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

func WithRole(roleName string) UserOption {
	return func(u *User) { u.Role = Role{Name: roleName} }
}
