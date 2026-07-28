package storage

import (
	"errors"
	"fmt"
)

type MapRepository struct {
	dict map[string]string
}

func NewMapRepository() *MapRepository {
	return &MapRepository{dict: make(map[string]string)}
}

func (r *MapRepository) Add(key string, url string) error {
	existingKey, err := r.GetKeyByURL(url)
	if err == nil && existingKey != "" {
		return &DuplicateError{key: existingKey, url: url}
	}
	r.dict[key] = url
	return nil
}

func (r *MapRepository) AddBatch(urls map[string]string) error {
	for key, url := range urls {
		existingKey, err := r.GetKeyByURL(url)
		if err == nil && existingKey != "" {
			return &DuplicateError{key: existingKey, url: url}
		}
		r.dict[key] = url
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

func (r *MapRepository) Ping() error {
	return nil
}

func (r *MapRepository) IsDuplicateError(err error) bool {
	var dupErr *DuplicateError
	return errors.As(err, &dupErr) && dupErr.Error() == "duplicate"
}
