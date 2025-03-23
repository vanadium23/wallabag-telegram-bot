package usecase

import (
	"sync"

	"github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"
)

const mxPool int = 64

type WallabotArticleUseCase struct {
	wc  *wallabag.WallabagClient
	mxs [mxPool]sync.Mutex
}

func NewWallabotArticleUseCase(wc *wallabag.WallabagClient) *WallabotArticleUseCase {
	return &WallabotArticleUseCase{
		wc:  wc,
		mxs: [mxPool]sync.Mutex{},
	}
}

func (wau *WallabotArticleUseCase) MarkRead(entryID int) (WallabotArticle, error) {
	wau.mxs[entryID%mxPool].Lock()
	defer wau.mxs[entryID%mxPool].Unlock()

	entry, err := wau.wc.FetchArticle(entryID)
	if err != nil {
		return WallabotArticle{}, err
	}

	if entry.IsArchived != 0 {
		return NewWallabotArticle(entry), nil
	}

	err = wau.wc.UpdateArticle(entryID, 1)
	if err != nil {
		return WallabotArticle{}, err
	}
	// todo: move to updatedArticle from wallabag client
	entry.IsArchived = 1
	return NewWallabotArticle(entry), nil
}

func (wau *WallabotArticleUseCase) MarkUnread(entryID int) (WallabotArticle, error) {
	wau.mxs[entryID%mxPool].Lock()
	defer wau.mxs[entryID%mxPool].Unlock()

	entry, err := wau.wc.FetchArticle(entryID)
	if err != nil {
		return WallabotArticle{}, err
	}

	if entry.IsArchived != 1 {
		return NewWallabotArticle(entry), nil
	}

	err = wau.wc.UpdateArticle(entryID, 0)
	if err != nil {
		return WallabotArticle{}, err
	}
	// todo: move to updatedArticle from wallabag client
	entry.IsArchived = 0
	return NewWallabotArticle(entry), nil
}

// func (wau *WallabotArticleUseCase) MarkScrolled(entryID int) (WallabotArticle, error)   {}
// func (wau *WallabotArticleUseCase) DeleteScrolled(entryID int) (WallabotArticle, error) {}
// func (wau *WallabotArticleUseCase) AddRating(entryID int) (WallabotArticle, error)      {}
// func (wau *WallabotArticleUseCase) DeleteRating(entryID int) (WallabotArticle, error)   {}
