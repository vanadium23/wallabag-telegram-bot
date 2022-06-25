package wallabag

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestWallabagClientCreateArticle(t *testing.T) {
	articleURL := "test"
	ClientID := "app_xxx"
	ClientSecret := "secret_xxx"
	Username := "unit"
	Password := "password"
	AccessToken := "access_token"

	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		switch path {
		case "/api/entries.json":
			var data WallabagEntry
			bearer := req.Header.Get("Authorization")
			if bearer != fmt.Sprintf("Bearer %s", AccessToken) {
				http.Error(rw, "Unauthorized", http.StatusUnauthorized)
				t.Errorf("No bearer token in request")
				return
			}

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
		case "/oauth/v2/token":
			data := WallabagOauthToken{
				AccessToken: "access_token",
				ExpiresIn:   24 * 60 * 60,
			}
			response, _ := json.Marshal(data)
			rw.Write(response)
		default:
			t.Errorf("Incorrect path %s", path)
		}
	}))
	// Close the server when test finishes
	defer server.Close()

	wallabagClient := NewWallabagClient(
		server.Client(),
		server.URL,
		ClientID,
		ClientSecret,
		Username,
		Password,
	)
	article, err := wallabagClient.CreateArticle(articleURL)
	if err != nil {
		t.Errorf("Unexpected error during %s", err)
	}
	if article.Url != articleURL {
		t.Errorf("Unexpected response %s", article)
	}
}

func TestWallabagClientFetchArticles(t *testing.T) {
	ClientID := "app_xxx"
	ClientSecret := "secret_xxx"
	Username := "unit"
	Password := "password"
	AccessToken := "access_token"

	articleURL := "test"
	page := 0
	perPage := 30
	archive := 0

	articles := []WallabagEntry{
		{
			Url: articleURL,
		},
	}

	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		switch path {
		case "/api/entries.json":
			bearer := req.Header.Get("Authorization")
			if bearer != fmt.Sprintf("Bearer %s", AccessToken) {
				http.Error(rw, "Unauthorized", http.StatusUnauthorized)
				t.Errorf("No bearer token in request")
				return
			}

			query := req.URL.Query()
			if query.Get("perPage") != strconv.Itoa(perPage) {
				t.Errorf("Incorrect perPage in query params")
			}

			if query.Get("page") != strconv.Itoa(page) {
				t.Errorf("Incorrect page in query params")
			}

			if query.Get("archive") != strconv.Itoa(archive) {
				t.Errorf("Incorrect archive in query params")
			}

			response, _ := json.Marshal(WallabagEntryResponse{
				Data: WallabagEntryResponseItems{
					Entries: articles,
				},
			})
			rw.Write(response)
		case "/oauth/v2/token":
			data := WallabagOauthToken{
				AccessToken: "access_token",
				ExpiresIn:   24 * 60 * 60,
			}
			response, _ := json.Marshal(data)
			rw.Write(response)
		default:
			t.Errorf("Incorrect path %s", path)
		}
	}))
	// Close the server when test finishes
	defer server.Close()

	wallabagClient := NewWallabagClient(
		server.Client(),
		server.URL,
		ClientID,
		ClientSecret,
		Username,
		Password,
	)
	articles, err := wallabagClient.FetchArticles(page, perPage, archive)
	if err != nil {
		t.Errorf("Unexpected error during %s", err)
	}
	if len(articles) == 0 {
		t.Errorf("No articles fetched")
	}
}
