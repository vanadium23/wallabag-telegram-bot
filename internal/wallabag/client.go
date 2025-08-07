package wallabag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type WallabagCreateEntry struct {
	Url  string `json:"url"`
	Tags string `json:"tags"`
}

type WallabagTag struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
	Slug  string `json:"slug"`
}

type WallabagEntry struct {
	Url         string        `json:"url"`
	ID          int           `json:"id,omitempty"`
	CreatedAt   *WallabagTime `json:"created_at"`
	UpdatedAt   *WallabagTime `json:"updated_at"`
	ArchivedAt  *WallabagTime `json:"archived_at"`
	StarredAt   *WallabagTime `json:"starred_at"`
	Content     string        `json:"content"`
	Title       string        `json:"title"`
	ReadingTime int           `json:"reading_time"`
	IsArchived  int           `json:"is_archived"`
	Tags        []WallabagTag `json:"tags"`
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

// WallabagTimeLayout is a variation of RFC3339 but without colons in
// the timezone delimiter, breaking the RFC
const WallabagTimeLayout = "2006-01-02T15:04:05-0700"

// WallabagTime overrides builtin time to allow for custom time parsing
type WallabagTime struct {
	time.Time
}

// UnmarshalJSON parses the custom date format wallabag returns
func (t *WallabagTime) UnmarshalJSON(buf []byte) (err error) {
	s := strings.Trim(string(buf), `"`)
	if s == "null" {
		t.Time = time.Time{}
		return err
	}
	t.Time, err = time.Parse(WallabagTimeLayout, s)
	if err != nil {
		t.Time = time.Time{}
		return err
	}
	return err
}

type WallabagClient struct {
	client      *http.Client
	baseURL     string
	defaultTags string

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
	defaultTags string,
) WallabagClient {
	return WallabagClient{
		client:             client,
		baseURL:            baseURL,
		clientID:           clientID,
		clientSecret:       clientSecret,
		username:           username,
		password:           password,
		accessTokenExpires: time.Now(),
		defaultTags:        defaultTags,
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

	newEntry := WallabagCreateEntry{
		Url:  articleURL,
		Tags: wc.defaultTags,
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

func (wc WallabagClient) FetchArticles(page int, perPage int, archive int, tags []string) ([]WallabagEntry, error) {
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

func (wc WallabagClient) FetchArticlesWithSince(page int, perPage int, archive int, since int64, tags []string) ([]WallabagEntry, error) {
	url := fmt.Sprintf("%s/api/entries.json?page=%d&perPage=%d&archive=%d&since=%d", wc.baseURL, page, perPage, archive, since)
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

func (wc WallabagClient) FetchArticle(entryID int) (WallabagEntry, error) {
	url := fmt.Sprintf("%s/api/entries/%d.json", wc.baseURL, entryID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return WallabagEntry{}, err
	}

	accessToken, err := wc.fetchAccessToken()
	if err != nil {
		return WallabagEntry{}, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := wc.client.Do(req)
	if err != nil {
		return WallabagEntry{}, err
	}
	var response WallabagEntry

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&response)

	return response, nil
}

func (wc WallabagClient) UpdateArticle(entryID int, archive int) (WallabagEntry, error) {
	updateEntry := WallabagUpdateEntryData{
		Archive: archive,
	}
	url := fmt.Sprintf("%s/api/entries/%d.json", wc.baseURL, entryID)
	data, _ := json.Marshal(updateEntry)
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(data))
	if err != nil {
		return WallabagEntry{}, err
	}

	accessToken, err := wc.fetchAccessToken()
	if err != nil {
		return WallabagEntry{}, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := wc.client.Do(req)
	if err != nil {
		return WallabagEntry{}, err
	}
	var response WallabagEntry

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&response)

	return response, nil
}

func (wc WallabagClient) AddTagsToArticle(entryID int, tags []string) (WallabagEntry, error) {
	data := map[string]string{
		"tags": strings.Join(tags, ","),
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return WallabagEntry{}, err
	}
	url := fmt.Sprintf("%s/api/entries/%d/tags.json", wc.baseURL, entryID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return WallabagEntry{}, err
	}

	accessToken, err := wc.fetchAccessToken()
	if err != nil {
		return WallabagEntry{}, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := wc.client.Do(req)
	if err != nil {
		return WallabagEntry{}, err
	}
	var response WallabagEntry

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&response)

	return response, nil
}
