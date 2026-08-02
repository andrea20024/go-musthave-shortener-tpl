package storage

import (
	"errors"
	"fmt"
	"sync"
)

type MapRepository struct {
	mu       sync.RWMutex
	dict     map[string]string
	userUrls map[string]map[string]string
	deleted  map[string]bool
}

func NewMapRepository() *MapRepository {
	return &MapRepository{
		dict:     make(map[string]string),
		userUrls: make(map[string]map[string]string),
		deleted:  make(map[string]bool),
	}
}

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

func (r *MapRepository) Ping() error {
	return nil
}

func (r *MapRepository) IsDuplicateError(err error) bool {
	var dupErr *DuplicateError
	return errors.As(err, &dupErr) && dupErr.Error() == "duplicate"
}

func (r *MapRepository) IsDeletedError(err error) bool {
	var delErr *DeletedError
	return errors.As(err, &delErr)
}
