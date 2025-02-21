package postservices

import (
	"github.com/google/uuid"
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

	posts, err = ps.UpdatePost(posts)
	if err != nil {
		return err
	}
	return nil
}

// IncrementReadingTime increments the reading time for a post
func (ps *PostSystem) ReadingTime(postID uuid.UUID, readingTime int) error {
	posts, err := ps.GetPostBy("id = ?", postID)
	if err != nil {
		return err
	}

	_, err = ps.UpdatePost(posts)
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
