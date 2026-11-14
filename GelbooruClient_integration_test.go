package gobooru

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestGelbooruSearch(t *testing.T) {
	if os.Getenv("GELBOORU_LIVE_TEST") != "1" {
		t.Skip("set GELBOORU_LIVE_TEST=1 to call the real API")
	}

	userID := os.Getenv("GELBOORU_USER_ID")
	apiKey := os.Getenv("GELBOORU_API_KEY")
	if userID == "" || apiKey == "" {
		t.Fatal("GELBOORU_USER_ID and GELBOORU_API_KEY must be set")
	}

	client := NewGelbooruClient(
		"gelbooru",
		"https://gelbooru.com/index.php",
		userID,
		apiKey,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	posts, err := client.Search().Tags("hatsune_miku").Get(ctx)
	if err != nil {
		t.Fatalf("live API request failed: %v", err)
	}

	if len(posts) == 0 {
		t.Fatal("API returned no posts for the test query")
	}

	post := posts[0]
	if post.ID <= 0 || post.FileUrl == "" {
		t.Fatalf("missing expected post fields: %+v", post)
	}

	t.Logf(
		"Received post: id=%d, size=%dx%d, tags=%d, url=%s",
		post.ID, post.Width, post.Height, len(post.Tags), post.FileUrl,
	)
}

func TestGelbooruAutocomplete(t *testing.T) {
	if os.Getenv("GELBOORU_LIVE_TEST") != "1" {
		t.Skip("set GELBOORU_LIVE_TEST=1 to call the real API")
	}

	userID := os.Getenv("GELBOORU_USER_ID")
	apiKey := os.Getenv("GELBOORU_API_KEY")
	if userID == "" || apiKey == "" {
		t.Fatal("GELBOORU_USER_ID and GELBOORU_API_KEY must be set")
	}

	client := NewGelbooruClient(
		"gelbooru",
		"https://gelbooru.com/index.php",
		userID,
		apiKey,
	)

	tags, err := client.Autocomplete("mari", 10)
	if err != nil {
		t.Fatalf("live API request failed: %v", err)
	}

	if len(tags) == 0 {
		t.Fatal("API returned no posts for the test query")
	}

	tag := tags[0]
	if tag.Value == "" {
		t.Fatalf("missing expected tag fields: %+v", tag)
	}

	for index, value := range tags {
		t.Logf("Index: %d, Value: %s\n", index, value.Value)
	}
}

func TestGelbooruDownload(t *testing.T) {
	if os.Getenv("GELBOORU_LIVE_TEST") != "1" {
		t.Skip("set GELBOORU_LIVE_TEST=1 to call the real API")
	}

	userID := os.Getenv("GELBOORU_USER_ID")
	apiKey := os.Getenv("GELBOORU_API_KEY")
	if userID == "" || apiKey == "" {
		t.Fatal("GELBOORU_USER_ID and GELBOORU_API_KEY must be set")
	}

	client := NewGelbooruClient(
		"gelbooru",
		"https://gelbooru.com/index.php",
		userID,
		apiKey,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	posts, err := client.Search().Tags("hatsune_miku").Limit(5).Get(ctx)
	if err != nil {
		t.Fatalf("live API request failed: %v", err)
	}

	client.Download(posts)
}
