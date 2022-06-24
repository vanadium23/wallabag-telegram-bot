package worker

import (
	"context"
	"time"

	"github.com/vanadium23/wallabag-telegram-bot/internal/articles"
	"github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"
)

type Wallabager interface {
	CreateArticle(articleURL string) (wallabag.WallabagEntry, error)
}

type Worker struct {
	wc             Wallabager
	ar             articles.ArticleRepository
	rescanInterval time.Duration

	diskQueue    chan saveURLRequest
	requestQueue chan saveURLRequest
}

type saveURLRequest struct {
	URL       string
	ChatID    int64
	MessageID int
}

func NewWorker(wc Wallabager, ar articles.ArticleRepository, rescanInterval time.Duration) Worker {
	return Worker{
		wc:             wc,
		ar:             ar,
		rescanInterval: rescanInterval,

		diskQueue:    make(chan saveURLRequest, 100),
		requestQueue: make(chan saveURLRequest, 100),
	}
}

func (w Worker) runQueueToDisk(ctx context.Context) {
	select {
	case r := <-w.diskQueue:
		w.ar.Insert(r.URL, r.ChatID, r.MessageID)
		w.requestQueue <- r
		break
	case <-ctx.Done():
		return
	}
}

func (w Worker) runQueueToWallabag(ctx context.Context, ackQueue chan saveURLRequest) {
	select {
	case r := <-w.requestQueue:
		article, err := w.wc.CreateArticle(r.URL)
		if err != nil {
			break
		}
		err = w.ar.Save(r.URL)
		if err != nil {
			break
		}
		ackQueue <- saveURLRequest{URL: article.Url, ChatID: r.ChatID, MessageID: r.MessageID}
		break
	case <-ctx.Done():
		return
	}
}

func (w Worker) rescanRepository(ctx context.Context) {
	for {
		timer := time.NewTimer(w.rescanInterval)

		articles, err := w.ar.FetchUnsaved()
		if err != nil {
			continue
		}

		for _, article := range articles {
			w.requestQueue <- saveURLRequest{URL: article.URL, ChatID: article.ChatID, MessageID: article.MessageID}
		}

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}
}

func (w Worker) SendToDisk(articleURL string, chatID int64, messageID int64) {
	w.diskQueue <- saveURLRequest{URL: articleURL, ChatID: chatID, MessageID: int(messageID)}
}

func (w Worker) Start(ctx context.Context, ackQueue chan saveURLRequest) {
	go w.runQueueToDisk(ctx)
	go w.runQueueToWallabag(ctx, ackQueue)
	go w.rescanRepository(ctx)
}
