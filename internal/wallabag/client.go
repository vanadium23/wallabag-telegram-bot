package wallabag

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type WallabagEntry struct {
	Url string `json:"url"`
}

type WallabagClient struct {
	client  *http.Client
	baseURL string
}

func NewWallabagClient(client *http.Client, baseURL string) WallabagClient {
	return WallabagClient{
		client:  client,
		baseURL: baseURL,
	}
}

func (wc WallabagClient) CreateArticle(articleURL string) (WallabagEntry, error) {
	var createdEntry WallabagEntry

	newEntry := WallabagEntry{
		Url: articleURL,
	}
	data, _ := json.Marshal(newEntry)
	resp, err := wc.client.Post(wc.baseURL+"/api/entries.json", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return createdEntry, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&createdEntry)
	if err != nil {
		return createdEntry, err
	}
	return createdEntry, err
}
