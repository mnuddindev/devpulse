package postservices

import (
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/models"
)

// IncrementViewCount increments the view count for a post and updates series analytics
func (ps *PostSystem) ViewCount(postID uuid.UUID) error {
	posts, err := ps.GetPostBy("id = ?", postID)
	if err != nil {
		return err
	}

	posts.ViewsCount++
	posts, err = ps.UpdatePost(posts)
	if err != nil {
		return err
	}

	if posts.SeriesID != nil {
		seriesAnalytics := posts.Series.Analytics
		seriesAnalytics.TotalViews++

		series := &models.Series{
			Analytics: seriesAnalytics,
		}

		_, err = ps.UpdateSeries(series)
		if err != nil {
			return err
		}
	}
	return nil
}

// IncrementReactionCount increments the reaction count for a post and updates series analytics
func (ps *PostSystem) ReactionCount(postID uuid.UUID) error {
	posts, err := ps.GetPostBy("id = ?", postID)
	if err != nil {
		return err
	}

	posts.LikesCount++
	posts, err = ps.UpdatePost(posts)
	if err != nil {
		return err
	}

	if posts.SeriesID != nil {
		seriesAnalytics := posts.Series.Analytics
		seriesAnalytics.TotalReactions++

		series := &models.Series{
			Analytics: seriesAnalytics,
		}

		_, err = ps.UpdateSeries(series)
		if err != nil {
			return err
		}
	}

	return nil
}
