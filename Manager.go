package goboorulib

import (
	"context"
	"fmt"
)

type Manager struct {
	clients map[string]BooruClient
}

// NewManager initializes an empty booru manager
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]BooruClient),
	}
}

// Register adds a client to the manager under a specific key (e.g., "gelbooru", "danbooru")
func (m *Manager) Register(name string, client BooruClient) {
	m.clients[name] = client
}

// Search is a unified facade method that delegates to the correct booru client
func (m *Manager) Search(ctx context.Context, booruName string, tags string, page int, limit int) ([]Post, error) {
	client, exists := m.clients[booruName]
	if !exists {
		return nil, fmt.Errorf("unsupported or unregistered booru: %s", booruName)
	}

	// Delegate the request to the specific client
	return client.SearchPosts(ctx, tags, page, limit)
}

func (m *Manager) Autocomplete(booruName string, query string, limit int) ([]Autocomplete, error) {
	client, exists := m.clients[booruName]
	if !exists {
		return nil, fmt.Errorf("unsupported or unregistered booru: %s", booruName)
	}

	return client.Autocomplete(query, limit)
}
