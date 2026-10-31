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
	MD5        string `json:"md5"`
}

type Attributes struct {
	Limit  int64 `json:"limit"`
	Offset int64 `json:"offset"`
	Count  int64 `json:"count"`
}

type GelbooruAutocompleteItem struct {
	Type      string `json:"type"`
	Label     string `json:"label"`
	Value     string `json:"value"`
	PostCount int    `json:"post_count,string"`
	Category  string `json:"category"`
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
func NewGelbooruClient(booruName string, baseURL string, userID string, apiKey string) *GelbooruClient {
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

type Search struct {
	client *GelbooruClient
	query  Query
	page   int
}

type Query struct {
	tags   string
	sortBy string
	limit  int
}

func (c *GelbooruClient) Search() *Search {
	return &Search{
		client: c,
		page:   0,
	}
}

func (s *Search) Tags(tags string) *Search {
	s.query.tags = tags
	return s
}

func (s *Search) SortBy(sort string) *Search {
	s.query.sortBy = sort
	return s
}

func (s *Search) Limit(limit int) *Search {
	s.query.limit = limit
	return s
}

func (s *Search) Get(ctx context.Context) ([]Post, error) {
	parsedURL, err := url.Parse(s.client.baseURL)

	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %w", err)
	}

	fmt.Print(s.query.limit)

	searchQuery := parsedURL.Query()
	searchQuery.Set("page", "dapi")
	searchQuery.Set("s", "post")
	searchQuery.Set("q", "index")
	searchQuery.Set("json", "1")
	searchQuery.Set("tags", s.query.tags+" "+s.query.sortBy)

	limit := s.query.limit
	if limit == 0 {
		limit = 100
	}

	searchQuery.Set("limit", strconv.Itoa(limit))

	// Gelbooru uses 0-index pages
	pid := max((s.page-1)*s.query.limit, 0)
	searchQuery.Set("pid", strconv.Itoa(pid))

	// Add the API credentials (very important)
	if s.client.userID != "" && s.client.apiKey != "" {
		searchQuery.Set("user_id", s.client.userID)
		searchQuery.Set("api_key", s.client.apiKey)
	}

	// Encode query
	parsedURL.RawQuery = searchQuery.Encode()

	// Build new request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)

	// Create request with context
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %w", err)
	}

	// Add header to prevent the API from blocking the library's requests
	req.Header.Set("User-Agent", "GoBooruLib/1.0 (GoLang)")

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code from %s: %d", s.client.booruName, resp.StatusCode)
	}

	// Keep the response available for both wrapped and top-level array formats.
	var response json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response from %s: %w", s.client.booruName, err)
	}

	var responseWrapper struct {
		Posts []GelbooruPost `json:"post"`
	}

	if err := json.Unmarshal(response, &responseWrapper); err != nil {
		if err := json.Unmarshal(response, &responseWrapper.Posts); err != nil {
			return nil, fmt.Errorf("failed to decode response from %s: %w", s.client.booruName, err)
		}
	}

	// Map raw posts into your clean, unified Post model
	var posts []Post
	for _, rp := range responseWrapper.Posts {
		posts = append(posts, Post{
			ID:          rp.ID,
			SourceBooru: s.client.booruName,
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

func (c *GelbooruClient) Autocomplete(query string, limit int) ([]TagSuggestion, error) {
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

	var results []GelbooruAutocompleteItem
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode autocomplete response from %s: %w", c.booruName, err)
	}

	var suggestions []TagSuggestion
	for _, rp := range results {
		suggestions = append(suggestions, TagSuggestion{
			Type:      rp.Type,
			Label:     rp.Label,
			Value:     rp.Value,
			PostCount: rp.PostCount,
			Category:  rp.Category,
		})
	}

	return suggestions, nil
}

// Download basically takes a list of posts and pipes them through a worker pool for concurrency.
func (c *GelbooruClient) Download(postsList []Post) {
	wp := DownloadWorkerPool{
		Posts:   postsList,
		Workers: 10, // 10 just to be safe. I don't want anyone getting banned at all just from using this library.
	}

	wp.Run()
}
