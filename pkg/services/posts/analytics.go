package postservices

import (
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/models"
)

// Increament increases count for a post
func (ps *PostSystem) IncreaseCount(postid uuid.UUID, ctype string) error {
	posts, err := ps.GetPostBy("id = ?", postid)
	if err != nil {
		return err
	}

	switch ctype {
	case "view":
		posts.PostAnalytics.ViewsCount++
	case "like":
		posts.PostAnalytics.LikesCount++
	case "comment":
		posts.PostAnalytics.CommentsCount++
	case "share":
		posts.PostAnalytics.ShareCount++
	case "bookmark":
		posts.PostAnalytics.BookmarksCount++
	case "complete":
		posts.PostAnalytics.CompleteCount++
	default:
		return nil
	}

	err = ps.Crud.Update(&posts, "id = ? AND ip_address = ?", []interface{}{postid, posts.PostAnalytics.IpAddress}, posts)
	if err != nil {
		return err
	}
	return nil
}

// IncrementSeriesCount increments the count for a series
func (ps *PostSystem) SeriesCount(seriesID uuid.UUID, ctype string) error {
	series, err := ps.GetSeriesBy("id = ?", seriesID)
	if err != nil {
		return err
	}

	switch ctype {
	case "view":
		series.Analytics.TotalViews++
	case "reaction":
		series.Analytics.TotalReactions++
	case "post":
		series.TotalPosts++
	default:
		return nil
	}

	_, err = ps.UpdateSeries(series)
	if err != nil {
		return err
	}
	return nil
}

// PostsAnalytics updates analytics data for all posts
func (ps *PostSystem) PostsAnalytics(input interface{}, ip string) error {
	var analytics models.PostAnalytics
	err := ps.Crud.Update(&analytics, "id = ? AND ip_address = ?", []interface{}{input, ip}, analytics)
	if err != nil {
		return err
	}
	return nil
}
