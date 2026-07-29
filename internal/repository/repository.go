package storage

type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Repository interface {
	Add(key string, url string, userID string) error
	AddBatch(urls map[string]string, userID string) error
	Get(key string) (string, error)
	GetKeyByURL(url string) (string, error)
	GetUserURLs(userID string) ([]UserURL, error)
	DeleteUserURLs(userID string, keys []string) error
	Ping() error
	IsDuplicateError(err error) bool
	IsDeletedError(err error) bool
}

type DuplicateError struct {
	key string
	url string
}

func (e *DuplicateError) Error() string {
	return "duplicate"
}

type DeletedError struct{}

func (e *DeletedError) Error() string { return "deleted" }
