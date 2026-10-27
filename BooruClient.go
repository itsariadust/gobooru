package goboorulib

import "context"

type BooruClient interface {
	SearchPosts(ctx context.Context, tags string, page int, limit int) ([]Post, error)
}
