package usecase

import (
	"strings"
	"time"

	"github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"
)

type ArticleUseCase interface {
	MarkRead(entryID int) (WallabotArticle, error)
	MarkUnread(entryID int) (WallabotArticle, error)
	MarkScrolled(entryID int) (WallabotArticle, error)
	// DeleteScrolled(entryID int) (WallabotArticle, error)
	AddRating(entryID int, rating string) (WallabotArticle, error)
	// DeleteRating(entryID int) (WallabotArticle, error)
	// Summarize(entryID int) (string, error)

	FindByID(entryID int) (WallabotArticle, error)
	SaveForLater(url string) (WallabotArticle, error)
	FindRandom(count int) ([]WallabotArticle, error)
	FindRecent(count int) ([]WallabotArticle, error)
	FindShort(count int) ([]WallabotArticle, error)

	GetStats() (WallabagStats, error)
}

type WallabagStats struct {
	TotalUnread       int `json:"total_unread"`
	ArchivedToday     int `json:"archived_today"`
	ArchivedLast7Days int `json:"archived_last_7_days"`
}

type WallabotArticle struct {
	ID          int
	IsRead      bool
	tags        []string
	Url         string
	Title       string
	Content     string
	CreatedAt   time.Time
	ReadingTime int

	HasRating bool
	Scrolled  bool
}

func NewWallabotArticle(entry wallabag.WallabagEntry) WallabotArticle {

	tags := make([]string, len(entry.Tags))
	scrolled := false
	hasRating := false
	for i := 0; i < len(entry.Tags); i++ {
		// TODO: calculate scrollable and rating
		tags[i] = entry.Tags[i].Label
		if tags[i] == "scrolled" {
			scrolled = true
		}
		_, ok := RatingFromString(tags[i])
		if ok {
			hasRating = true
		}
	}

	return WallabotArticle{
		ID:          entry.ID,
		IsRead:      entry.IsArchived != 0,
		tags:        tags,
		Url:         entry.Url,
		Content:     entry.Content,
		Title:       entry.Title,
		CreatedAt:   entry.CreatedAt.Time,
		ReadingTime: entry.ReadingTime,
		Scrolled:    scrolled,
		HasRating:   hasRating,
	}
}

func (wa WallabotArticle) PublicTags() string {
	return strings.Join(wa.tags, ", ")
}
