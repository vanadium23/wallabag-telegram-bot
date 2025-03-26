package usecase

import (
	"errors"
	"log"
	"sync"

	"math/rand/v2"

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

	entry, err := wau.wc.AddTagsToArticle(entryID, []string{"scrolled"})

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

	entry, err := wau.wc.AddTagsToArticle(entryID, []string{rating})

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
