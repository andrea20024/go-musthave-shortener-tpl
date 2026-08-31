// Package storage provides in-memory implementation of the Repository interface.
//
// This file contains the MapRepository struct which implements all methods of
// the Repository interface using in-memory maps protected by sync.RWMutex.
// Suitable for development and testing.
package storage

import (
	"errors"
	"fmt"
	"sync"
)

// MapRepository implements the Repository interface using in-memory maps.
// It is suitable for development and testing purposes.
type MapRepository struct {
	mu       sync.RWMutex
	dict     map[string]string
	userUrls map[string]map[string]string
	deleted  map[string]bool
}

// NewMapRepository creates a new MapRepository with initialized maps.
func NewMapRepository() *MapRepository {
	return &MapRepository{
		dict:     make(map[string]string),
		userUrls: make(map[string]map[string]string),
		deleted:  make(map[string]bool),
	}
}

// Add stores a URL mapping, checking for duplicates first.
func (r *MapRepository) Add(key string, url string, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for existingKey, val := range r.dict {
		if val == url && !r.deleted[existingKey] {
			return &DuplicateError{key: existingKey, url: url}
		}
	}

	r.dict[key] = url
	if r.userUrls[userID] == nil {
		r.userUrls[userID] = make(map[string]string)
	}
	r.userUrls[userID][key] = url
	r.deleted[key] = false
	return nil
}

// AddBatch stores multiple URL mappings, checking for duplicates.
func (r *MapRepository) AddBatch(urls map[string]string, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, url := range urls {
		for existingKey, val := range r.dict {
			if val == url && !r.deleted[existingKey] {
				return &DuplicateError{key: existingKey, url: url}
			}
		}
		r.dict[key] = url
		if r.userUrls[userID] == nil {
			r.userUrls[userID] = make(map[string]string)
		}
		r.userUrls[userID][key] = url
		r.deleted[key] = false
	}
	return nil
}

// Get retrieves the original URL by short URL key, returning DeletedError if deleted.
func (r *MapRepository) Get(key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.deleted[key] {
		return "", &DeletedError{}
	}
	val, ok := r.dict[key]
	if ok {
		return val, nil
	}
	return "", fmt.Errorf("key not found: %s", key)
}

// GetKeyByURL finds the short URL key for a given original URL.
func (r *MapRepository) GetKeyByURL(url string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for key, val := range r.dict {
		if val == url && !r.deleted[key] {
			return key, nil
		}
	}
	return "", fmt.Errorf("url not found: %s", url)
}

// GetUserURLs retrieves all non-deleted URLs for a specific user.
func (r *MapRepository) GetUserURLs(userID string) ([]UserURL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	urls := make([]UserURL, 0)
	for key, orig := range r.userUrls[userID] {
		if !r.deleted[key] {
			urls = append(urls, UserURL{
				ShortURL:    key,
				OriginalURL: orig,
			})
		}
	}
	return urls, nil
}

// DeleteUserURLs marks the specified URLs as deleted for the given user.
func (r *MapRepository) DeleteUserURLs(userID string, keys []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.userUrls[userID] == nil {
		return nil
	}
	for _, key := range keys {
		if _, exists := r.userUrls[userID][key]; exists {
			r.deleted[key] = true
		}
	}
	return nil
}

// Ping checks if the repository is available (always returns nil for in-memory).
func (r *MapRepository) Ping() error {
	return nil
}

// IsDuplicateError checks if the error is a DuplicateError.
func (r *MapRepository) IsDuplicateError(err error) bool {
	var dupErr *DuplicateError
	return errors.As(err, &dupErr) && dupErr.Error() == "duplicate"
}

// IsDeletedError checks if the error is a DeletedError.
func (r *MapRepository) IsDeletedError(err error) bool {
	var delErr *DeletedError
	return errors.As(err, &delErr)
}

// Shutdown gracefully closes the in-memory storage.
// No data persistence is needed as MapRepository is in-memory only.
func (r *MapRepository) Shutdown() error {
	return nil
}

// Stats returns the count of non-deleted URLs and distinct users.
func (r *MapRepository) Stats() (int, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	urls := 0
	for key := range r.dict {
		if !r.deleted[key] {
			urls++
		}
	}

	return urls, len(r.userUrls), nil
}
