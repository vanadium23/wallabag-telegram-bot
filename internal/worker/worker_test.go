package worker

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/vanadium23/wallabag-telegram-bot/internal/articles"
	"github.com/vanadium23/wallabag-telegram-bot/internal/wallabag"
)

type WallabagClientMock struct{}

func (w WallabagClientMock) CreateArticle(articleURL string) (wallabag.WallabagEntry, error) {
	return wallabag.WallabagEntry{
		Url: articleURL,
	}, nil
}

func TestWorker(t *testing.T) {
	articleURL := "test"
	chatID := int64(1)
	messageID := int64(2)
	ackQueue := make(chan SaveURLRequest, 1)
	ctx := context.Background()

	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Errorf("%v", err)
	}

	articleRepo, err := articles.NewArticleRepo(database)
	if err != nil {
		t.Errorf("%v", err)
	}

	wc := WallabagClientMock{}

	w := NewWorker(wc, articleRepo, time.Hour)
	w.Start(ctx, ackQueue)
	w.SendToDisk(articleURL, chatID, messageID)
	ackMsg := <-ackQueue
	if ackMsg.URL != articleURL {
		t.Fatalf("Unexpected URL in message from ackQueue %s != %s", ackMsg.URL, articleURL)
	}
	if ackMsg.ChatID != chatID {
		t.Fatalf("Unexpected chatID in message from ackQueue %d != %d", ackMsg.ChatID, chatID)
	}

	count, err := articleRepo.CountArticleByURL(articleURL)
	if err != nil {
		t.Fatalf("Error when count article")
	}

	if count != 1 {
		t.Fatalf("Expected %d article, found %d", 1, count)
	}
}

func TestWorkerRescan(t *testing.T) {
	articleURL := "test"
	chatID := int64(1)
	messageID := int64(2)
	ackQueue := make(chan SaveURLRequest, 1)
	ctx := context.Background()

	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Errorf("%v", err)
	}

	articleRepo, err := articles.NewArticleRepo(database)
	if err != nil {
		t.Errorf("%v", err)
	}
	articleRepo.Insert(articleURL, chatID, int(messageID))

	wc := WallabagClientMock{}

	w := NewWorker(wc, articleRepo, time.Second)
	w.Start(ctx, ackQueue)
	ackMsg := <-ackQueue
	if ackMsg.URL != articleURL {
		t.Fatalf("Unexpected URL in message from ackQueue %s != %s", ackMsg.URL, articleURL)
	}
	if ackMsg.ChatID != chatID {
		t.Fatalf("Unexpected chatID in message from ackQueue %d != %d", ackMsg.ChatID, chatID)
	}

	count, err := articleRepo.CountArticleByURL(articleURL)
	if err != nil {
		t.Fatalf("Error when count article")
	}

	if count != 1 {
		t.Fatalf("Expected %d article, found %d", 1, count)
	}
}
