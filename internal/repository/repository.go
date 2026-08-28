// Package storage provides interface and implementations for URL storage.
//
// The package defines the Repository interface which abstracts the underlying
// storage mechanism. Three implementations are available:
//   - MapRepository: in-memory storage using sync.RWMutex
//   - FileRepository: file-based storage with event log (append-only JSON file)
//   - dbRepository: PostgreSQL-backed storage using pgx driver
//
// Error types DuplicateError and DeletedError are used to signal specific
// conditions during URL operations.
package storage

import "fmt"

// UserURL represents a mapping between a short URL and its original URL
// for a specific user.
// generate:reset
type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// Repository defines the interface for URL storage operations.
// All methods are expected to be safe for concurrent use by multiple goroutines.
type Repository interface {
	// Add stores a mapping between a short URL key and the original URL.
	// If the original URL already exists with a different key, returns DuplicateError.
	Add(key string, url string, userID string) error

	// AddBatch stores multiple short-to-original URL mappings in a single batch.
	// If any original URL already exists, returns DuplicateError.
	AddBatch(urls map[string]string, userID string) error

	// Get retrieves the original URL by its short URL key.
	// Returns DeletedError if the URL has been deleted by the user.
	Get(key string) (string, error)

	// GetKeyByURL finds the short URL key for a given original URL.
	// Returns an error if the original URL is not found.
	GetKeyByURL(url string) (string, error)

	// GetUserURLs retrieves all URLs shortened by a specific user.
	GetUserURLs(userID string) ([]UserURL, error)

	// DeleteUserURLs marks the specified URLs as deleted for the given user.
	DeleteUserURLs(userID string, keys []string) error

	// Ping checks if the storage backend is available and responsive.
	Ping() error

	// IsDuplicateError reports whether err is a DuplicateError.
	IsDuplicateError(err error) bool

	// IsDeletedError reports whether err is a DeletedError.
	IsDeletedError(err error) bool

	// Shutdown gracefully closes the storage backend.
	Shutdown() error

	// Stats returns the number of non-deleted URLs and the number of distinct users.
	Stats() (int, int, error)
}

// DuplicateError is returned when attempting to add a URL that already exists
// in the storage with a different short URL key.
type DuplicateError struct {
	key string
	url string
}

// Error implements the error interface for DuplicateError, returning "duplicate".
func (e *DuplicateError) Error() string {
	return "duplicate"
}

// DeletedError is returned when attempting to access a URL that has been
// marked as deleted by its owner.
type DeletedError struct{}

// Error implements the error interface for DeletedError, returning "deleted".
func (e *DeletedError) Error() string { return "deleted" }

// ErrKeyNotFound is returned when a short URL key cannot be found in storage.
var ErrKeyNotFound = fmt.Errorf("key not found")
