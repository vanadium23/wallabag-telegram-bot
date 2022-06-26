package wallabag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WallabagEntry struct {
	Url string `json:"url"`
	ID  int    `json:"id,omitempty"`
}

type WallabagOauthToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type WallabagEntryResponseItems struct {
	Entries []WallabagEntry `json:"items"`
}

type WallabagEntryResponse struct {
	Data WallabagEntryResponseItems `json:"_embedded"`
}

type WallabagUpdateEntryData struct {
	Archive int `json:"archive"`
}

type WallabagClient struct {
	client  *http.Client
	baseURL string

	// oAuth params
	clientID     string
	clientSecret string
	username     string
	password     string

	// token state
	accessToken        string
	accessTokenExpires time.Time
}

func NewWallabagClient(
	client *http.Client,
	baseURL string,
	clientID string,
	clientSecret string,
	username string,
	password string,
) WallabagClient {
	return WallabagClient{
		client:             client,
		baseURL:            baseURL,
		clientID:           clientID,
		clientSecret:       clientSecret,
		username:           username,
		password:           password,
		accessTokenExpires: time.Now(),
	}
}

func (wc *WallabagClient) fetchAccessToken() (string, error) {
	if time.Now().Before(wc.accessTokenExpires) && wc.accessToken != "" {
		return wc.accessToken, nil
	}
	queryParams := fmt.Sprintf("?grant_type=password&client_id=%s&client_secret=%s&username=%s&password=%s",
		wc.clientID, wc.clientSecret, wc.username, wc.password)
	resp, err := wc.client.Get(wc.baseURL + "/oauth/v2/token" + queryParams)
	if err != nil {
		return "", err
	}
	var data WallabagOauthToken
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", err
	}
	wc.accessTokenExpires = time.Now().Local().Add(time.Second * time.Duration(data.ExpiresIn))
	wc.accessToken = data.AccessToken
	return wc.accessToken, err
}

func (wc WallabagClient) CreateArticle(articleURL string) (WallabagEntry, error) {
	var createdEntry WallabagEntry

	newEntry := WallabagEntry{
		Url: articleURL,
	}
	data, _ := json.Marshal(newEntry)
	req, err := http.NewRequest("POST", wc.baseURL+"/api/entries.json", bytes.NewBuffer(data))

	if err != nil {
		return createdEntry, err
	}
	accessToken, err := wc.fetchAccessToken()
	if err != nil {
		return createdEntry, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := wc.client.Do(req)
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

func (wc WallabagClient) FetchArticles(page int, perPage int, archive int) ([]WallabagEntry, error) {
	url := fmt.Sprintf("%s/api/entries.json?page=%d&perPage=%d&archive=%d", wc.baseURL, page, perPage, archive)
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}
	accessToken, err := wc.fetchAccessToken()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := wc.client.Do(req)
	if err != nil {
		return nil, err
	}
	var response WallabagEntryResponse

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return response.Data.Entries, err
}

func (wc WallabagClient) UpdateArticle(entryID int, archive int) error {
	updateEntry := WallabagUpdateEntryData{
		Archive: archive,
	}
	url := fmt.Sprintf("%s/api/entries/%d.json", wc.baseURL, entryID)
	data, _ := json.Marshal(updateEntry)
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	accessToken, err := wc.fetchAccessToken()
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	_, err = wc.client.Do(req)
	if err != nil {
		return err
	}
	return nil
}
