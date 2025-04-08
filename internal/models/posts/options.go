package models

import (
	"strings"
	"time"

	user "github.com/mnuddindev/devpulse/internal/models/user"
)

func WithPostTitle(title string) PostsOption {
	return func(p *Posts) {
		trimmed := strings.TrimSpace(title)
		p.Title = trimmed
	}
}

func WithPostSlug(slug string) PostsOption {
	return func(p *Posts) {
		slug = strings.ToLower(strings.TrimSpace(slug))
		p.Slug = slug
	}
}

func WithPostContent(content string) PostsOption {
	return func(p *Posts) {
		trimmed := strings.TrimSpace(content)
		p.Content = trimmed
	}
}

func WithPostStatus(status string) PostsOption {
	return func(p *Posts) {
		p.Status = status
		p.Published = (status == "published" || status == "public")
		if p.Published && p.PublishedAt == nil {
			now := time.Now()
			p.PublishedAt = &now
		} else if !p.Published {
			p.PublishedAt = nil
		}
	}
}

func WithPostMentions(mentions []user.User) PostsOption {
	return func(p *Posts) {
		p.Mentions = mentions
	}
}

func WithPostCoAuthors(coAuthors []user.User) PostsOption {
	return func(p *Posts) {
		p.CoAuthors = coAuthors
	}
}

// PostAnalytics
func WithViewsCount(delta int) PostsOption {
	return func(pa *Posts) {
		pa.PostAnalytics.ViewsCount += delta
		if pa.PostAnalytics.ViewsCount < 0 {
			pa.PostAnalytics.ViewsCount = 0
		}
	}
}

func WithCommentsCount(delta int) PostsOption {
	return func(pa *Posts) {
		pa.PostAnalytics.CommentsCount += delta
		if pa.PostAnalytics.CommentsCount < 0 {
			pa.PostAnalytics.CommentsCount = 0
		}
	}
}

func WithReactionsCount(delta int) PostsOption {
	return func(pa *Posts) {
		pa.PostAnalytics.ReactionsCount += delta
		if pa.PostAnalytics.ReactionsCount < 0 {
			pa.PostAnalytics.ReactionsCount = 0
		}
	}
}

func WithBookmarksCount(delta int) PostsOption {
	return func(pa *Posts) {
		pa.PostAnalytics.BookmarksCount += delta
		if pa.PostAnalytics.BookmarksCount < 0 {
			pa.PostAnalytics.BookmarksCount = 0
		}
	}
}

func WithReadTime(minutes int) PostsOption {
	return func(pa *Posts) {
		pa.PostAnalytics.ReadTime = minutes
	}
}
