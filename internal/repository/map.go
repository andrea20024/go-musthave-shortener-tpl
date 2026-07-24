package storage

type MapRepository struct {
	dict map[string]string
}

func NewMapRepository() *MapRepository {
	return &MapRepository{dict: make(map[string]string)}
}

func (r *MapRepository) Add(key string, url string) {
	r.dict[key] = url
}

func (r *MapRepository) Get(key string) string {
	val, ok := r.dict[key]
	if ok {
		return val
	}
	return ""
}

func (r *MapRepository) Ping() error {
	return nil
}
