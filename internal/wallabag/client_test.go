package wallabag

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

/*
Как бы я писал тест? Что должен делать клиент?
- Сохранять статью.
- Менеджить токен к wallabag. <-- это внутреннее состояние

Нужна ли мне структура? Что там будет храниться?
- Настройки для доступа.
- Текущий токен?

Соответственно, сценарий для теста такой:
1. Создаём клиент к wallabag.
2. Передаём статью для сохранения.
3. Получаем, что статья успешно сохранена.
*/

func TestWallabagClient(t *testing.T) {
	articleURL := "test"
	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		path := req.URL.String()
		switch path {
		case "/api/entries.json":
			var data WallabagEntry
			err := json.NewDecoder(req.Body).Decode(&data)
			if err != nil {
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}
			if data.Url != articleURL {
				t.Errorf("Provided article is not equal %s == %s", data.Url, articleURL)
			}
			response, _ := json.Marshal(data)
			rw.Write(response)
		default:
			t.Errorf("Incorrect path %s", path)
		}
	}))
	// Close the server when test finishes
	defer server.Close()

	wallabagClient := NewWallabagClient(server.Client(), server.URL)
	article, err := wallabagClient.CreateArticle(articleURL)
	if err != nil {
		t.Errorf("Unexpected error during %s", err)
	}
	if article.Url != articleURL {
		t.Errorf("Unexpected response %s", article)
	}
}
