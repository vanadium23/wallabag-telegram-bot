package usecase

import "github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"

type ArticleUseCases interface {
	MarkRead(entryID int) (WallabotArticle, error)
	MarkUnread(entryID int) (WallabotArticle, error)
	MarkScrolled(entryID int) (WallabotArticle, error)
	DeleteScrolled(entryID int) (WallabotArticle, error)
	AddRating(entryID int) (WallabotArticle, error)
	DeleteRating(entryID int) (WallabotArticle, error)
	// Summarize(entryID int) (string, error)
}

type WallabotArticle struct {
	ID     int
	IsRead bool
	tags   []string
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
