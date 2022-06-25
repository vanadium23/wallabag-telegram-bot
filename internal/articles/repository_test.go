package articles

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestArticleRepository(t *testing.T) {
	articleURL := "test"
	chatID := int64(1)
	messageID := 2

	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Errorf("%v", err)
	}

	articleRepo, err := NewArticleRepo(database)
	if err != nil {
		t.Errorf("%v", err)
	}

	err = articleRepo.Insert(articleURL, chatID, messageID)
	if err != nil {
		t.Errorf("%v", err)
	}

	articles, err := articleRepo.FetchUnsaved()
	if err != nil {
		t.Errorf("%v", err)
	}

	if len(articles) != 1 {
		t.Errorf("Expected one unsaved article, found: %d", len(articles))
	}

	err = articleRepo.Save(articleURL)
	if err != nil {
		t.Errorf("%v", err)
	}

	count, err := articleRepo.CountArticleByURL(articleURL)
	if err != nil {
		t.Errorf("%v", err)
	}

	if count != 1 {
		t.Errorf("Expected one article by URL %s, found: %d", articleURL, count)
	}
}
