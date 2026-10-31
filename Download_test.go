package goboorulib

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestMD5Verification(t *testing.T) {
	const contents = "valid file"
	expectedHash := fmt.Sprintf("%x", md5.Sum([]byte(contents)))
	for _, tt := range []struct {
		name     string
		exists   bool
		contents string
		want     bool
	}{
		{name: "missing file"},
		{name: "matching file", exists: true, contents: contents, want: true},
		{name: "mismatched file", exists: true, contents: "corrupt file"},
		{name: "empty mismatched file", exists: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "file")
			if tt.exists {
				if err := os.WriteFile(filename, []byte(tt.contents), 0600); err != nil {
					t.Fatal(err)
				}
			}

			verified, err := md5Verification(filename, expectedHash)
			if err != nil || verified != tt.want {
				t.Fatalf("md5Verification = (%v, %v), want (%v, nil)", verified, err, tt.want)
			}
			if tt.want {
				got, err := os.ReadFile(filename)
				if err != nil || string(got) != contents {
					t.Fatalf("verified file = %q, %v; want %q, nil", got, err, contents)
				}
			} else if _, err := os.Stat(filename); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("unverified file should not exist, got %v", err)
			}
		})
	}
}

func TestMD5VerificationReadError(t *testing.T) {
	filename := t.TempDir()
	verified, err := md5Verification(filename, "invalid")
	if err == nil || verified {
		t.Fatalf("md5Verification on directory = (%v, %v), want (false, error)", verified, err)
	}
	if _, err := os.Stat(filename); err != nil {
		t.Fatalf("verification error should leave path untouched: %v", err)
	}
}

func TestDownloadWorkerPoolMD5(t *testing.T) {
	t.Chdir(t.TempDir())
	const contents = "valid file"
	expectedHash := fmt.Sprintf("%x", md5.Sum([]byte(contents)))
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path == "/matching" || r.URL.Path == "/unreadable" {
			t.Errorf("unexpected download for %s", r.URL.Path)
		}
		fmt.Fprint(w, contents)
	}))
	defer server.Close()

	if err := os.WriteFile("matching", []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("mismatched", []byte("corrupt file"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("unreadable", 0700); err != nil {
		t.Fatal(err)
	}
	pool := DownloadWorkerPool{Workers: 1}
	for _, filename := range []string{"matching", "unreadable", "mismatched", "missing"} {
		pool.Posts = append(pool.Posts, Post{FileUrl: server.URL + "/" + filename, MD5: expectedHash})
	}
	finished := make(chan struct{})
	go func() {
		pool.Run()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("worker pool did not complete all posts")
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("download requests = %d, want 2", got)
	}
	for _, filename := range []string{"matching", "mismatched", "missing"} {
		got, err := os.ReadFile(filename)
		if err != nil || string(got) != contents {
			t.Errorf("%s = %q, %v; want %q, nil", filename, got, err, contents)
		}
	}
}
