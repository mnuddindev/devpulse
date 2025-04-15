package models

import (
	"strings"
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

// Tag Options
func WithTagName(name string) TagOption {
	return func(t *Tag) { t.Name = strings.TrimSpace(name) }
}

func WithTagIsApproved(isapproved bool) TagOption {
	return func(t *Tag) {
		t.IsApproved = isapproved
	}
}

func WithTagSlug(slug string) TagOption {
	return func(t *Tag) { t.Slug = strings.ToLower(strings.TrimSpace(slug)) }
}

func WithTagDescription(desc string) TagOption {
	return func(t *Tag) { t.Description = strings.TrimSpace(desc) }
}

func WithTagShortDescription(desc string) TagOption {
	return func(t *Tag) { t.ShortDescription = strings.TrimSpace(desc) }
}

func WithTagIconURL(url string) TagOption {
	return func(t *Tag) { t.IconURL = url }
}

func WithTagBackgroundURL(url string) TagOption {
	return func(t *Tag) { t.BackgroundURL = url }
}

func WithTagTextColor(color string) TagOption {
	return func(t *Tag) { t.TextColor = color }
}

func WithTagBackgroundColor(color string) TagOption {
	return func(t *Tag) { t.BackgroundColor = color }
}

func WithTagIsFeatured(featured bool) TagOption {
	return func(t *Tag) { t.IsFeatured = featured }
}

func WithTagIsModerated(moderated bool) TagOption {
	return func(t *Tag) { t.IsModerated = moderated }
}

func WithTagRules(rules string) TagOption {
	return func(t *Tag) { t.Rules = strings.TrimSpace(rules) }
}

func WithTagModerators(moderators []user.User) TagOption {
	return func(t *Tag) { t.Moderators = moderators }
}

// TagAnalytics Options
func WithDailyViews(delta int) TagAnalyticsOption {
	return func(ta *TagAnalytics) {
		ta.DailyViews += delta
		if ta.DailyViews < 0 {
			ta.DailyViews = 0
		}
	}
}

func WithWeeklyViews(delta int) TagAnalyticsOption {
	return func(ta *TagAnalytics) {
		ta.WeeklyViews += delta
		if ta.WeeklyViews < 0 {
			ta.WeeklyViews = 0
		}
	}
}

func WithMonthlyViews(delta int) TagAnalyticsOption {
	return func(ta *TagAnalytics) {
		ta.MonthlyViews += delta
		if ta.MonthlyViews < 0 {
			ta.MonthlyViews = 0
		}
	}
}

func WithDailyFollowers(delta int) TagAnalyticsOption {
	return func(ta *TagAnalytics) {
		ta.DailyFollowers += delta
		if ta.DailyFollowers < 0 {
			ta.DailyFollowers = 0
		}
	}
}

func WithWeeklyFollowers(delta int) TagAnalyticsOption {
	return func(ta *TagAnalytics) {
		ta.WeeklyFollowers += delta
		if ta.WeeklyFollowers < 0 {
			ta.WeeklyFollowers = 0
		}
	}
}

func WithMonthlyFollowers(delta int) TagAnalyticsOption {
	return func(ta *TagAnalytics) {
		ta.MonthlyFollowers += delta
		if ta.MonthlyFollowers < 0 {
			ta.MonthlyFollowers = 0
		}
	}
}

// Series Options
func WithSeriesTitle(title string) SeriesOption {
	return func(s *Series) {
		s.Title = strings.TrimSpace(title)
	}
}

func WithSeriesSlug(slug string) SeriesOption {
	return func(s *Series) {
		s.Slug = strings.ToLower(strings.TrimSpace(slug))
	}
}

func WithSeriesDescription(description string) SeriesOption {
	return func(s *Series) {
		s.Description = strings.TrimSpace(description)
	}
}

func WithSeriesCoverImageURL(url string) SeriesOption {
	return func(s *Series) {
		s.CoverImageURL = strings.TrimSpace(url)
	}
}

func WithSeriesAuthorID(authorID uuid.UUID) SeriesOption {
	return func(s *Series) {
		s.AuthorID = authorID
	}
}

func WithSeriesIsPublished(isPublished bool) SeriesOption {
	return func(s *Series) {
		s.IsPublished = isPublished
	}
}

func WithSeriesTotalPostsDelta(delta int) SeriesOption {
	return func(s *Series) {
		s.TotalPosts += delta
		if s.TotalPosts < 0 {
			s.TotalPosts = 0
		}
	}
}

// SeriesPost Options
func WithSeriesPostSID(seriesID uuid.UUID) SeriesPostOption {
	return func(sp *SeriesPost) {
		sp.SeriesID = seriesID
	}
}

func WithSeriesPostID(postID uuid.UUID) SeriesPostOption {
	return func(sp *SeriesPost) {
		sp.PostID = postID
	}
}

func WithSeriesPostPosition(position int) SeriesPostOption {
	return func(sp *SeriesPost) {
		sp.Position = position
	}
}

// SeriesAnalytics Options
func WithSeriesSeriesAnalyticsID(seriesID uuid.UUID) SeriesAnalyticsOption {
	return func(sa *SeriesAnalytics) {
		sa.SeriesID = seriesID
	}
}

func WithSeriesAnalyticsTotalViews(totalViews int) SeriesAnalyticsOption {
	return func(sa *SeriesAnalytics) {
		sa.TotalViews = totalViews
	}
}

func WithSeriesAnalyticsTotalReactions(totalReactions int) SeriesAnalyticsOption {
	return func(sa *SeriesAnalytics) {
		sa.TotalReactions = totalReactions
	}
}

func WithSeriesAnalyticsAverageReadTime(readTime float64) SeriesAnalyticsOption {
	return func(sa *SeriesAnalytics) {
		sa.AverageReadTime = readTime
	}
}

func WithSeriesAnalyticsCompletionRate(rate float64) SeriesAnalyticsOption {
	return func(sa *SeriesAnalytics) {
		sa.CompletionRate = rate
	}
}
