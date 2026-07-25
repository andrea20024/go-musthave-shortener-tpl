package storage

type Repository interface {
	Add(key string, url string) error
	AddBatch(urls map[string]string) error
	Get(key string) string
	GetKeyByURL(url string) string
	Ping() error
	IsDuplicateError(err error) bool
}

type DuplicateError struct {
	key string
	url string
}

func (e *DuplicateError) Error() string {
	return "duplicate"
}
