package models

import (
	"time"

	"github.com/google/uuid"
	user "github.com/mnuddindev/devpulse/internal/models/user"
)

// Core Content Fields
func WithTitle(title string) PostsOption {
	return func(p *Posts) {
		p.Title = title
	}
}

func WithSlug(slug string) PostsOption {
	return func(p *Posts) {
		p.Slug = slug
	}
}

func WithContent(content string) PostsOption {
	return func(p *Posts) {
		p.Content = content
	}
}

func WithExcerpt(excerpt string) PostsOption {
	return func(p *Posts) {
		p.Excerpt = excerpt
	}
}

func WithFeaturedImageURL(url string) PostsOption {
	return func(p *Posts) {
		p.FeaturedImageURL = url
	}
}

func WithStatus(status string) PostsOption {
	return func(p *Posts) {
		p.Status = status
	}
}

func WithPublishingStatus(status string) PostsOption {
	return func(p *Posts) {
		p.PublishingStatus = status
	}
}

func WithContentFormat(format string) PostsOption {
	return func(p *Posts) {
		p.ContentFormat = format
	}
}

func WithCanonicalURL(url string) PostsOption {
	return func(p *Posts) {
		p.CanonicalURL = url
	}
}

// SEO & Social Metadata
func WithMetaTitle(title string) PostsOption {
	return func(p *Posts) {
		p.MetaTitle = title
	}
}

func WithMetaDescription(desc string) PostsOption {
	return func(p *Posts) {
		p.MetaDescription = desc
	}
}

func WithSEOKeywords(keywords string) PostsOption {
	return func(p *Posts) {
		p.SEOKeywords = keywords
	}
}

func WithOGTitle(title string) PostsOption {
	return func(p *Posts) {
		p.OGTitle = title
	}
}

func WithOGDescription(desc string) PostsOption {
	return func(p *Posts) {
		p.OGDescription = desc
	}
}

func WithOGImageURL(url string) PostsOption {
	return func(p *Posts) {
		p.OGImageURL = url
	}
}

func WithTwitterTitle(title string) PostsOption {
	return func(p *Posts) {
		p.TwitterTitle = title
	}
}

func WithTwitterDescription(desc string) PostsOption {
	return func(p *Posts) {
		p.TwitterDescription = desc
	}
}

func WithTwitterImageURL(url string) PostsOption {
	return func(p *Posts) {
		p.TwitterImageURL = url
	}
}

// Collaboration & Review System
func WithAuthorID(authorID uuid.UUID) PostsOption {
	return func(p *Posts) {
		p.AuthorID = authorID
	}
}

func WithSeriesID(seriesID *uuid.UUID) PostsOption {
	return func(p *Posts) {
		p.SeriesID = seriesID
	}
}

func WithEditedAt(editedAt *time.Time) PostsOption {
	return func(p *Posts) {
		p.EditedAt = editedAt
	}
}

func WithLastEditedByID(userID *uuid.UUID) PostsOption {
	return func(p *Posts) {
		p.LastEditedByID = userID
	}
}

func WithNeedsReview(needsReview bool) PostsOption {
	return func(p *Posts) {
		p.NeedsReview = needsReview
	}
}

func WithReviewedByID(userID *uuid.UUID) PostsOption {
	return func(p *Posts) {
		p.ReviewedByID = userID
	}
}

func WithReviewedAt(reviewedAt *time.Time) PostsOption {
	return func(p *Posts) {
		p.ReviewedAt = reviewedAt
	}
}

// Publishing Fields
func WithPublished(published bool) PostsOption {
	return func(p *Posts) {
		p.Published = published
	}
}

func WithPublishedAt(publishedAt *time.Time) PostsOption {
	return func(p *Posts) {
		p.PublishedAt = publishedAt
	}
}

// Relationships
func WithTags(tags []Tag) PostsOption {
	return func(p *Posts) {
		p.Tags = tags
	}
}

func WithComments(comments []Comment) PostsOption {
	return func(p *Posts) {
		p.Comments = comments
	}
}

func WithReactions(reactions []Reaction) PostsOption {
	return func(p *Posts) {
		p.Reactions = reactions
	}
}

func WithBookmarks(bookmarks []Bookmark) PostsOption {
	return func(p *Posts) {
		p.Bookmarks = bookmarks
	}
}

func WithMentions(mentions []user.User) PostsOption {
	return func(p *Posts) {
		p.Mentions = mentions
	}
}

func WithCoAuthors(coAuthors []user.User) PostsOption {
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
