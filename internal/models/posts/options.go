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
func WithViewsCount(delta int) PostAnalyticsOption {
	return func(pa *PostAnalytics) {
		pa.ViewsCount += delta
		if pa.ViewsCount < 0 {
			pa.ViewsCount = 0
		}
	}
}

func WithCommentsCount(delta int) PostAnalyticsOption {
	return func(pa *PostAnalytics) {
		pa.CommentsCount += delta
		if pa.CommentsCount < 0 {
			pa.CommentsCount = 0
		}
	}
}

func WithReactionsCount(delta int) PostAnalyticsOption {
	return func(pa *PostAnalytics) {
		pa.ReactionsCount += delta
		if pa.ReactionsCount < 0 {
			pa.ReactionsCount = 0
		}
	}
}

func WithBookmarksCount(delta int) PostAnalyticsOption {
	return func(pa *PostAnalytics) {
		pa.BookmarksCount += delta
		if pa.BookmarksCount < 0 {
			pa.BookmarksCount = 0
		}
	}
}

func WithReadTime(minutes int) PostAnalyticsOption {
	return func(pa *PostAnalytics) {
		pa.ReadTime = minutes
	}
}
