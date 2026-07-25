package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
)

type Event struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type FileRepository struct {
	filename string
	dict     map[string]string
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
	repo := &FileRepository{filename: filename, dict: make(map[string]string)}
	events, err := repo.loadEvents()
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		repo.dict[event.ShortURL] = event.OriginalURL
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

func (r *FileRepository) Add(key string, url string) error {
	existingKey := r.GetKeyByURL(url)
	if existingKey != "" {
		return &DuplicateError{key: existingKey, url: url}
	}
	r.dict[key] = url
	event := Event{ShortURL: key, OriginalURL: url}
	if err := r.writeFile(&event); err != nil {
		return err
	}
	return nil
}

func (r *FileRepository) AddBatch(urls map[string]string) error {
	for key, url := range urls {
		existingKey := r.GetKeyByURL(url)
		if existingKey != "" {
			return &DuplicateError{key: existingKey, url: url}
		}
		r.dict[key] = url
		event := Event{ShortURL: key, OriginalURL: url}
		if err := r.writeFile(&event); err != nil {
			return err
		}
	}
	return nil
}

func (r *FileRepository) Get(key string) string {
	return r.dict[key]
}

func (r *FileRepository) GetKeyByURL(url string) string {
	for key, val := range r.dict {
		if val == url {
			return key
		}
	}
	return ""
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
