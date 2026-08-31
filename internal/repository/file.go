// Package storage provides file-based implementation of the Repository interface.
//
// This file contains:
//   - Event        — JSON-serializable representation of a URL mapping event.
//   - FileRepository — persistent storage backed by an append-only JSON file.
//   - Consumer     — read-only scanner used to replay events from file storage.
//
// FileRepository implements thread safety via sync.RWMutex. Every Add, AddBatch,
// DeleteUserURLs, and GetUserURLs operation is immediately flushed to disk as a
// JSON event line.
package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

// Event represents a single URL mapping event for file storage persistence.
type Event struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	IsDeleted   bool   `json:"is_deleted"`
}

// FileRepository implements the Repository interface using an append-only JSON file.
type FileRepository struct {
	mu       sync.RWMutex
	filename string
	dict     map[string]string
	userUrls map[string]map[string]string
	deleted  map[string]bool
}

// Consumer provides sequential read access to events from a file-based storage.
type Consumer struct {
	file    *os.File
	scanner *bufio.Scanner
}

// NewConsumer opens a file and creates a Consumer for reading events sequentially.
func NewConsumer(filename string) (*Consumer, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return &Consumer{file: file, scanner: bufio.NewScanner(file)}, nil
}

// ReadEvent reads the next JSON event from the file. Returns io.EOF at end of file.
func (c *Consumer) ReadEvent() (*Event, error) {
	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	data := c.scanner.Bytes()
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// Close closes the underlying file handle.
func (c *Consumer) Close() error {
	return c.file.Close()
}

// NewFileRepository creates a FileRepository and loads all existing events from the file.
func NewFileRepository(filename string) (*FileRepository, error) {
	repo := &FileRepository{
		filename: filename,
		dict:     make(map[string]string),
		userUrls: make(map[string]map[string]string),
		deleted:  make(map[string]bool),
	}
	events, err := repo.loadEvents()
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		repo.dict[event.ShortURL] = event.OriginalURL
		if event.UserID != "" {
			if repo.userUrls[event.UserID] == nil {
				repo.userUrls[event.UserID] = make(map[string]string)
			}
			repo.userUrls[event.UserID][event.ShortURL] = event.OriginalURL
		}
		if event.IsDeleted {
			repo.deleted[event.ShortURL] = true
		}
	}
	return repo, nil
}

func (r *FileRepository) loadEvents() ([]Event, error) {
	file, err := os.OpenFile(r.filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var events []Event
	for scanner.Scan() {
		data := scanner.Bytes()
		var event Event
		if err := json.Unmarshal(data, &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events, scanner.Err()
}

// Add stores a URL mapping, checking for duplicates first.
func (r *FileRepository) Add(key string, url string, userID string) error {
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
	event := Event{ShortURL: key, OriginalURL: url, UserID: userID, IsDeleted: false}
	if err := r.writeFile(&event); err != nil {
		return err
	}
	return nil
}

// AddBatch stores multiple URL mappings, checking for duplicates.
func (r *FileRepository) AddBatch(urls map[string]string, userID string) error {
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
		event := Event{ShortURL: key, OriginalURL: url, UserID: userID, IsDeleted: false}
		if err := r.writeFile(&event); err != nil {
			return err
		}
	}
	return nil
}

// Get retrieves the original URL by short URL key, returning DeletedError if deleted.
func (r *FileRepository) Get(key string) (string, error) {
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
func (r *FileRepository) GetKeyByURL(url string) (string, error) {
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
func (r *FileRepository) GetUserURLs(userID string) ([]UserURL, error) {
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
func (r *FileRepository) DeleteUserURLs(userID string, keys []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.userUrls[userID] == nil {
		return nil
	}
	for _, key := range keys {
		if _, exists := r.userUrls[userID][key]; exists {
			r.deleted[key] = true
			event := Event{ShortURL: key, IsDeleted: true}
			if err := r.writeFile(&event); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetDict returns a copy of the internal key-value map.
func (r *FileRepository) GetDict() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string, len(r.dict))
	for k, v := range r.dict {
		result[k] = v
	}
	return result
}

// Ping checks if the storage file exists and is accessible.
func (r *FileRepository) Ping() error {
	file, err := os.OpenFile(r.filename, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

// IsDuplicateError checks if the error is a DuplicateError.
func (r *FileRepository) IsDuplicateError(err error) bool {
	var dupErr *DuplicateError
	return errors.As(err, &dupErr) && dupErr.Error() == "duplicate"
}

// IsDeletedError checks if the error is a DeletedError.
func (r *FileRepository) IsDeletedError(err error) bool {
	var delErr *DeletedError
	return errors.As(err, &delErr)
}

// Shutdown is a no-op. FileRepository persists data on every write operation,
// so there is nothing to flush or close on shutdown.
func (r *FileRepository) Shutdown() error {
	return nil
}

// Stats returns the count of non-deleted URLs and distinct users.
func (r *FileRepository) Stats() (int, int, error) {
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

// writeFile appends a single event to the storage file.
func (r *FileRepository) writeFile(event *Event) error {
	file, err := os.OpenFile(r.filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	data, err := json.Marshal(&event)
	if err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}
	if err := writer.WriteByte('\n'); err != nil {
		return err
	}
	return writer.Flush()
}
