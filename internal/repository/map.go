package storage

var dict = make(map[string]string)

func Add(key string, url string) {
	dict[key] = url
}
func Get(key string) string {
	val, ok := dict[key]
	if ok {
		return val
	}
	return ""
}
