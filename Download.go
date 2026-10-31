package goboorulib

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
)

/*
DownloadWorkerPool is a struct that sets up the settings for the worker pool.
The worker pool accepts a list of [Post] which would be funneled into a channel for the workers to consume.
*/
type DownloadWorkerPool struct {
	Posts   []Post
	Workers int
	channel chan Post
	wg      sync.WaitGroup
}

/*
downloadWorker is the worker that handles the image download.
It checks first whether the file is there or not so there would be duplicate protection.
If the file is missing, download it. If it exists and the MD5 hashes are correct then it finishes the task.
If it exists but there are MD5 mismatches then it would replace that file with a new download.
*/
func (dwp *DownloadWorkerPool) downloadWorker() error {
	var workerErr error
	for task := range dwp.channel {
		err := func() error {
			defer dwp.wg.Done()
			filename := extractFilename(task.FileUrl)
			verified, err := md5Verification(filename, task.MD5)
			if err != nil {
				return fmt.Errorf("failed to verify file: %w", err)
			}
			if verified {
				return nil
			}

			out, err := os.Create(filename)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer out.Close()

			resp, err := http.Get(task.FileUrl)
			if err != nil {
				return fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("bad status: %s", resp.Status)
			}

			if _, err := io.Copy(out, resp.Body); err != nil {
				return fmt.Errorf("failed to save file payload: %w", err)
			}
			return nil
		}()
		if err != nil && workerErr == nil {
			workerErr = err
		}
	}

	return workerErr
}

/*
Run is the method to start up the workers based on the worker pool's settings.
It spawns a number of workers based on [DownloadWorkerPool]'s Worker count.
*/
func (dwp *DownloadWorkerPool) Run() {
	dwp.channel = make(chan Post, len(dwp.Posts))

	for i := 0; i < dwp.Workers; i++ {
		go dwp.downloadWorker()
	}

	dwp.wg.Add(len(dwp.Posts))
	for _, post := range dwp.Posts {
		dwp.channel <- post
	}

	close(dwp.channel)

	dwp.wg.Wait()
}

/*
md5Verification verifies the
*/
func md5Verification(path string, fileToDownloadHash string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	hash := md5.New()
	_, err = io.Copy(hash, file)
	closeErr := file.Close()
	if err != nil {
		return false, err
	}
	if closeErr != nil {
		return false, closeErr
	}

	localHash := hex.EncodeToString(hash.Sum(nil))

	if localHash != fileToDownloadHash {
		if err := os.Remove(path); err != nil {
			return false, err
		}
		return false, nil
	}

	return true, nil
}

func extractFilename(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		fmt.Printf("Failed to parse URL %s: %v\n", rawURL, err)
	}

	return path.Base(parsedURL.Path)
}
