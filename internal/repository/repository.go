package storage

type Repository interface {
	Add(key string)
	Get(ket string) string
}
