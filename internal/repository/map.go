package storage

import (
	"errors"
	"fmt"
)

type MapRepository struct {
	dict     map[string]string
	userUrls map[string]map[string]string
}

func NewMapRepository() *MapRepository {
	return &MapRepository{
		dict:     make(map[string]string),
		userUrls: make(map[string]map[string]string),
	}
}

func (r *MapRepository) Add(key string, url string, userID string) error {
	existingKey, err := r.GetKeyByURL(url)
	if err == nil && existingKey != "" {
		return &DuplicateError{key: existingKey, url: url}
	}
	r.dict[key] = url
	if r.userUrls[userID] == nil {
		r.userUrls[userID] = make(map[string]string)
	}
	r.userUrls[userID][key] = url
	return nil
}

func (r *MapRepository) AddBatch(urls map[string]string, userID string) error {
	for key, url := range urls {
		existingKey, err := r.GetKeyByURL(url)
		if err == nil && existingKey != "" {
			return &DuplicateError{key: existingKey, url: url}
		}
		r.dict[key] = url
		if r.userUrls[userID] == nil {
			r.userUrls[userID] = make(map[string]string)
		}
		r.userUrls[userID][key] = url
	}
	return nil
}

func (r *MapRepository) Get(key string) (string, error) {
	val, ok := r.dict[key]
	if ok {
		return val, nil
	}
	return "", fmt.Errorf("key not found: %s", key)
}

func (r *MapRepository) GetKeyByURL(url string) (string, error) {
	for key, val := range r.dict {
		if val == url {
			return key, nil
		}
	}
	return "", fmt.Errorf("url not found: %s", url)
}

func (r *MapRepository) GetUserURLs(userID string) ([]UserURL, error) {
	urls := make([]UserURL, 0)
	for key, orig := range r.userUrls[userID] {
		urls = append(urls, UserURL{
			ShortURL:    key,
			OriginalURL: orig,
		})
	}
	return urls, nil
}

func (r *MapRepository) Ping() error {
	return nil
}

func (r *MapRepository) IsDuplicateError(err error) bool {
	var dupErr *DuplicateError
	return errors.As(err, &dupErr) && dupErr.Error() == "duplicate"
}
