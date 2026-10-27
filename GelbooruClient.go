package goboorulib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Gelbooru-format struct
type GelbooruPost struct {
	ID         int64  `json:"id"`
	FileUrl    string `json:"file_url"`
	PreviewUrl string `json:"preview_url"`
	Tags       string `json:"tags"`
	Score      int    `json:"score"`
	Rating     string `json:"rating"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

type Attributes struct {
	Limit  int64 `json:"limit"`
	Offset int64 `json:"offset"`
	Count  int64 `json:"count"`
}

// Client parameters
type GelbooruClient struct {
	baseURL    string
	httpClient *http.Client
	booruName  string
	apiKey     string
	userID     string
}

// The Gelbooru client is for any imageboard that uses the Gelbooru Beta 0.2/0.2.5 system.
// Anything else needs their own unique implementation (e.g Danbooru).
func NewGelbooruClient(booruName, baseURL string, userID, apiKey string) *GelbooruClient {
	return &GelbooruClient{
		baseURL:   baseURL,
		booruName: booruName,
		userID:    userID,
		apiKey:    apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SearchPosts takes four parameters: context, tags (use camel case!), the page, and the limit.
// No defaults are given. All will be given via configuration from applications.
func (c *GelbooruClient) SearchPosts(ctx context.Context, tags string, page int, limit int) ([]Post, error) {
	parsedURL, err := url.Parse(c.baseURL)

	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %w", err)
	}

	// Set query params
	query := parsedURL.Query()
	query.Set("page", "dapi")
	query.Set("s", "post")
	query.Set("q", "index")
	query.Set("json", "1")
	query.Set("tags", tags)
	query.Set("limit", strconv.Itoa(limit))

	// Gelbooru uses 0-index pages
	pid := max((page-1)*limit, 0)
	query.Set("pid", strconv.Itoa(pid))

	// Add the API credentials (very important)
	if c.userID != "" && c.apiKey != "" {
		query.Set("user_id", c.userID)
		query.Set("api_key", c.apiKey)
	}

	// Encode query
	parsedURL.RawQuery = query.Encode()

	// Build new request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)

	// Create request with context
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %w", err)
	}

	// Add header to prevent the API from blocking the library's requests
	req.Header.Set("User-Agent", "GoBooruLib/1.0 (GoLang)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code from %s: %d", c.booruName, resp.StatusCode)
	}

	// Decode the wrapped posts; metadata and unused post fields are ignored.
	var responseWrapper struct {
		Posts []GelbooruPost `json:"post"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseWrapper); err != nil {
		return nil, fmt.Errorf("failed to decode response from %s: %w", c.booruName, err)
	}

	// Map raw posts into your clean, unified Post model
	var posts []Post
	for _, rp := range responseWrapper.Posts {
		posts = append(posts, Post{
			ID:          rp.ID,
			SourceBooru: c.booruName,
			FileUrl:     rp.FileUrl,
			PreviewUrl:  rp.PreviewUrl,
			Tags:        strings.Fields(rp.Tags), // Instantly splits space-separated tags into a slice
			Score:       rp.Score,
			Rating:      rp.Rating,
			Width:       rp.Width,
			Height:      rp.Height,
		})
	}

	return posts, nil
}

func (c *GelbooruClient) Autocomplete(query string, limit int) ([]Autocomplete, error) {
	parsedURL, err := url.Parse(c.baseURL)

	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %w", err)
	}

	autoCompleteQuery := parsedURL.Query()
	autoCompleteQuery.Set("page", "autocomplete2")
	autoCompleteQuery.Set("term", query)
	autoCompleteQuery.Set("type", "tag_query")
	autoCompleteQuery.Set("limit", strconv.Itoa(limit))

	// Encode query
	parsedURL.RawQuery = autoCompleteQuery.Encode()

	// Build new request
	req, err := http.NewRequest(http.MethodGet, parsedURL.String(), nil)

	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %w", err)
	}

	// Add header to prevent the API from blocking the library's requests
	req.Header.Set("User-Agent", "GoBooruLib/1.0 (GoLang)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code from %s: %d", c.booruName, resp.StatusCode)
	}

	var results []Autocomplete
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode autocomplete response from %s: %w", c.booruName, err)
	}

	return results, nil
}
