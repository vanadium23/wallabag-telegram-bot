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

	SaveForLater(url string) (WallabotArticle, error)
	FindRandom(count int) ([]WallabotArticle, error)
	FindRecent(count int) ([]WallabotArticle, error)
	// FindShort(count int) ([]WallabotArticle, error)
}

type WallabotArticle struct {
	ID          int
	IsRead      bool
	tags        []string
	Url         string
	Title       string
	CreatedAt   time.Time
	ReadingTime int

	HasRating bool
	Scrolled  bool
}

func NewWallabotArticle(entry wallabag.WallabagEntry) WallabotArticle {

	tags := make([]string, len(entry.Tags))
	for i := 0; i < len(entry.Tags); i++ {
		// TODO: calculate scrollable and rating
		tags[i] = entry.Tags[i].Label
	}

	return WallabotArticle{
		ID:     entry.ID,
		IsRead: entry.IsArchived != 0,
		tags:   tags,
	}
}

func (wa WallabotArticle) PublicTags() string {
	return strings.Join(wa.tags, ",")
}
