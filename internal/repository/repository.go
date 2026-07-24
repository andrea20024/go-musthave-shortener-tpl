package storage

type Repository interface {
	Add(key string, url string)
	AddBatch(urls map[string]string)
	Get(key string) string
	Ping() error
}
