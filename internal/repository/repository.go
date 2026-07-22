package storage

type Repository interface {
	Add(key string, url string)
	Get(key string) string
	Ping() error
}
