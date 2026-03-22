package storage

type Repository interface {
	Add(key string)
	Get(key string) string
}
