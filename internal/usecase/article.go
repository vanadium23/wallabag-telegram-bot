package usecase

import (
	"errors"
	"log"
	"sync"

	"math/rand/v2"
	"time"

	"github.com/vanadium23/wallabag-telegram-bot/internal/tagging"
	"github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"
)

const mxPool int = 64

type WallabotArticleUseCase struct {
	wc     wallabag.WallabagClient
	tagger tagging.Tagger
	mxs    [mxPool]sync.Mutex
}

func NewWallabotArticleUseCase(wc wallabag.WallabagClient, tagger tagging.Tagger) *WallabotArticleUseCase {
	return &WallabotArticleUseCase{
		wc:     wc,
		tagger: tagger,
		mxs:    [mxPool]sync.Mutex{},
	}
}

func (wau *WallabotArticleUseCase) MarkRead(entryID int) (WallabotArticle, error) {
	wau.mxs[entryID%mxPool].Lock()
	defer wau.mxs[entryID%mxPool].Unlock()

	entry, err := wau.wc.UpdateArticle(entryID, 1)
	if err != nil {
		return WallabotArticle{}, err
	}
	return NewWallabotArticle(entry), nil
}

func (wau *WallabotArticleUseCase) MarkUnread(entryID int) (WallabotArticle, error) {
	wau.mxs[entryID%mxPool].Lock()
	defer wau.mxs[entryID%mxPool].Unlock()

	entry, err := wau.wc.UpdateArticle(entryID, 0)
	if err != nil {
		return WallabotArticle{}, err
	}
	return NewWallabotArticle(entry), nil
}

func (wau *WallabotArticleUseCase) MarkScrolled(entryID int) (WallabotArticle, error) {
	wau.mxs[entryID%mxPool].Lock()
	defer wau.mxs[entryID%mxPool].Unlock()

	_, err := wau.wc.AddTagsToArticle(entryID, []string{"scrolled"})

	if err != nil {
		return WallabotArticle{}, err
	}

	entry, err := wau.wc.UpdateArticle(entryID, 1)
	if err != nil {
		return WallabotArticle{}, err
	}
	return NewWallabotArticle(entry), nil
}

// func (wau *WallabotArticleUseCase) DeleteScrolled(entryID int) (WallabotArticle, error) {}

func (wau *WallabotArticleUseCase) AddRating(entryID int, rating string) (WallabotArticle, error) {
	_, ok := RatingFromString(rating)
	if !ok {
		return WallabotArticle{}, errors.New("rating invalid")
	}

	wau.mxs[entryID%mxPool].Lock()
	defer wau.mxs[entryID%mxPool].Unlock()

	_, err := wau.wc.AddTagsToArticle(entryID, []string{rating})

	if err != nil {
		return WallabotArticle{}, err
	}

	entry, err := wau.wc.UpdateArticle(entryID, 1)
	if err != nil {
		return WallabotArticle{}, err
	}
	return NewWallabotArticle(entry), nil
}

// func (wau *WallabotArticleUseCase) DeleteRating(entryID int) (WallabotArticle, error)   {}

func (wau *WallabotArticleUseCase) SaveForLater(url string) (WallabotArticle, error) {
	entry, err := wau.wc.CreateArticle(url)
	if err != nil {
		return WallabotArticle{}, err
	}
	tags, err := wau.tagger.GuessTags(entry.Title, entry.Content)
	if err != nil {
		log.Printf("error on tagging: %v\n", err)
	}
	if tags != nil {
		entry, err = wau.wc.AddTagsToArticle(entry.ID, tags)
		if err != nil {
			return WallabotArticle{}, err
		}
	}
	return NewWallabotArticle(entry), err
}

func (wau *WallabotArticleUseCase) FindRandom(count int) ([]WallabotArticle, error) {
	entries, err := wau.wc.FetchArticles(1, 100, 0, nil)
	if err != nil {
		return nil, err
	}
	total := min(len(entries), count)
	articles := make([]WallabotArticle, total)
	for i := 0; i < total; i++ {
		articles[i] = NewWallabotArticle(entries[rand.IntN(len(entries))])
	}
	return articles, nil
}

func (wau *WallabotArticleUseCase) FindRecent(count int) ([]WallabotArticle, error) {
	entries, err := wau.wc.FetchArticles(1, count, 0, nil)
	if err != nil {
		return nil, err
	}
	total := min(len(entries), count)
	articles := make([]WallabotArticle, total)
	for i := 0; i < total; i++ {
		articles[i] = NewWallabotArticle(entries[i])
	}
	return articles, nil
}

func (wau *WallabotArticleUseCase) FindShort(count int) ([]WallabotArticle, error) {
	entries, err := wau.wc.FetchArticles(1, 100, 0, []string{"short"})
	if err != nil {
		return nil, err
	}
	total := min(len(entries), count)
	articles := make([]WallabotArticle, total)
	for i := 0; i < total; i++ {
		articles[i] = NewWallabotArticle(entries[i])
	}
	return articles, nil
}

func (wau *WallabotArticleUseCase) FindByID(entryID int) (WallabotArticle, error) {
	wau.mxs[entryID%mxPool].Lock()
	defer wau.mxs[entryID%mxPool].Unlock()

	entry, err := wau.wc.FetchArticle(entryID)
	if err != nil {
		return WallabotArticle{}, err
	}
	return NewWallabotArticle(entry), nil
}

func (wau *WallabotArticleUseCase) GetStats() (WallabagStats, error) {
	var stats WallabagStats

	// Get total unread articles (archive=0 means unread)
	unreadEntries, err := wau.wc.FetchArticlesWithSince(1, 1000, 0, 0, nil, "metadata")
	if err != nil {
		return stats, err
	}
	stats.TotalUnread = len(unreadEntries)

	// Calculate time boundaries for filtering
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	sevenDaysAgo := today.AddDate(0, 0, -7)
	tomorrow := today.AddDate(0, 0, 1)

	// Convert to Unix timestamps for the API 'since' parameter
	sevenDaysAgoUnix := sevenDaysAgo.Unix()

	// Get archived articles from the last 7 days using the 'since' parameter
	recentArchivedEntries, err := wau.wc.FetchArticlesWithSince(1, 1000, 1, sevenDaysAgoUnix, nil, "metadata")
	if err != nil {
		return stats, err
	}

	// Count articles archived today and in the last 7 days
	for _, entry := range recentArchivedEntries {
		if entry.ArchivedAt != nil {
			archivedDate := entry.ArchivedAt.Time

			// Check if archived today
			if archivedDate.After(today) && archivedDate.Before(tomorrow) {
				stats.ArchivedToday++
			}

			// Check if archived in last 7 days (including today)
			if archivedDate.After(sevenDaysAgo) {
				stats.ArchivedLast7Days++
			}
		}
	}

	// Get articles added in the last 7 days (both archived and unarchived)
	// First get unarchived articles created since 7 days ago
	recentUnreadEntries, err := wau.wc.FetchArticlesWithSince(1, 1000, 0, sevenDaysAgoUnix, nil, "metadata")
	if err != nil {
		return stats, err
	}

	// Then get archived articles created since 7 days ago (different from the archived articles query above)
	recentAllArchivedEntries, err := wau.wc.FetchArticlesWithSince(1, 1000, 1, sevenDaysAgoUnix, nil, "metadata")
	if err != nil {
		return stats, err
	}

	// Count articles added (created) in the last 7 days
	for _, entry := range recentUnreadEntries {
		if entry.CreatedAt != nil {
			createdDate := entry.CreatedAt.Time
			if createdDate.After(sevenDaysAgo) {
				stats.AddedLast7Days++
			}
		}
	}

	for _, entry := range recentAllArchivedEntries {
		if entry.CreatedAt != nil {
			createdDate := entry.CreatedAt.Time
			if createdDate.After(sevenDaysAgo) {
				stats.AddedLast7Days++
			}
		}
	}

	return stats, nil
}
