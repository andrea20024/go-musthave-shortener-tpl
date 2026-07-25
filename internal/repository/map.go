package storage

import "errors"

type MapRepository struct {
	dict map[string]string
}

func NewMapRepository() *MapRepository {
	return &MapRepository{dict: make(map[string]string)}
}

func (r *MapRepository) Add(key string, url string) error {
	existingKey := r.GetKeyByURL(url)
	if existingKey != "" {
		return &DuplicateError{key: existingKey, url: url}
	}
	r.dict[key] = url
	return nil
}

func (r *MapRepository) AddBatch(urls map[string]string) error {
	for key, url := range urls {
		existingKey := r.GetKeyByURL(url)
		if existingKey != "" {
			return &DuplicateError{key: existingKey, url: url}
		}
		r.dict[key] = url
	}
	return nil
}

func (r *MapRepository) Get(key string) string {
	val, ok := r.dict[key]
	if ok {
		return val
	}
	return ""
}

func (r *MapRepository) GetKeyByURL(url string) string {
	for key, val := range r.dict {
		if val == url {
			return key
		}
	}
	return ""
}

func (r *MapRepository) Ping() error {
	return nil
}

func (r *MapRepository) IsDuplicateError(err error) bool {
	var dupErr *DuplicateError
	return errors.As(err, &dupErr) && dupErr.Error() == "duplicate"
}
