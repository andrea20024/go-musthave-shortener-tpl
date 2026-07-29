package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type Event struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	IsDeleted   bool   `json:"is_deleted"`
}

type FileRepository struct {
	filename string
	dict     map[string]string
	userUrls map[string]map[string]string
	deleted  map[string]bool
}

type Consumer struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewConsumer(filename string) (*Consumer, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return &Consumer{file: file, scanner: bufio.NewScanner(file)}, nil
}

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

func (c *Consumer) Close() error {
	return c.file.Close()
}

func NewFileRepository(filename string) (*FileRepository, error) {
	repo := &FileRepository{
		filename: filename, dict: make(map[string]string),
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

func (r *FileRepository) Add(key string, url string, userID string) error {
	existingKey, err := r.GetKeyByURL(url)
	if err == nil && existingKey != "" {
		return &DuplicateError{key: existingKey, url: url}
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

func (r *FileRepository) AddBatch(urls map[string]string, userID string) error {
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
		r.deleted[key] = false
		event := Event{ShortURL: key, OriginalURL: url, UserID: userID, IsDeleted: false}
		if err := r.writeFile(&event); err != nil {
			return err
		}
	}
	return nil
}

func (r *FileRepository) Get(key string) (string, error) {
	if r.deleted[key] {
		return "", &DeletedError{}
	}
	val, ok := r.dict[key]
	if ok {
		return val, nil
	}
	return "", fmt.Errorf("key not found: %s", key)
}

func (r *FileRepository) GetKeyByURL(url string) (string, error) {
	for key, val := range r.dict {
		if val == url && !r.deleted[key] {
			return key, nil
		}
	}
	return "", fmt.Errorf("url not found: %s", url)
}

func (r *FileRepository) GetUserURLs(userID string) ([]UserURL, error) {
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

func (r *FileRepository) DeleteUserURLs(userID string, keys []string) error {
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

func (r *FileRepository) GetDict() map[string]string {
	return r.dict
}

func (r *FileRepository) Ping() error {
	file, err := os.OpenFile(r.filename, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

func (r *FileRepository) IsDuplicateError(err error) bool {
	var dupErr *DuplicateError
	return errors.As(err, &dupErr) && dupErr.Error() == "duplicate"
}

func (r *FileRepository) IsDeletedError(err error) bool {
	var delErr *DeletedError
	return errors.As(err, &delErr)
}

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
