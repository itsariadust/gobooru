package gobooru

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestSearchGetResponseFormats(t *testing.T) {
	const post = `{"id":42,"file_url":"https://example.com/file.jpg","preview_url":"https://example.com/preview.jpg","tags":"cat  blue_eyes","score":7,"rating":"s","width":640,"height":480}`
	want := []Post{{
		ID: 42, SourceBooru: "testbooru",
		FileUrl: "https://example.com/file.jpg", PreviewUrl: "https://example.com/preview.jpg",
		Tags: []string{"cat", "blue_eyes"}, Score: 7, Rating: "s", Width: 640, Height: 480,
	}}

	for _, tt := range []struct {
		name    string
		body    string
		want    []Post
		wantErr bool
	}{
		{name: "wrapped", body: `{"@attributes":{"count":1},"post":[` + post + `]}`, want: want},
		{name: "array", body: "[" + post + "]", want: want},
		{name: "empty wrapped", body: `{"post":[]}`},
		{name: "empty array", body: `[]`},
		{name: "malformed JSON", body: `[{`, wantErr: true},
		{name: "invalid wrapped post", body: `{"post":[{"id":"invalid"}]}`, wantErr: true},
		{name: "invalid array post", body: `[{"id":"invalid"}]`, wantErr: true},
		{name: "unexpected scalar", body: `42`, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()

			client := NewGelbooruClient("testbooru", server.URL, "", "")
			posts, err := client.Search().Get(context.Background())
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "failed to decode response from testbooru") {
					t.Fatalf("expected response decoding error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if !reflect.DeepEqual(posts, tt.want) {
				t.Fatalf("posts = %+v, want %+v", posts, tt.want)
			}
		})
	}
}
