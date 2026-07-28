package config

type Config struct {
	Host      string `env:"SERVER_ADDRESS"`
	BaseURL   string `env:"BASE_URL"`
	FilePath  string `env:"FILE_STORAGE_PATH"`
	DB        string `env:"DATABASE_DSN"`
	StoreType string
}

func InitConfig() *Config {
	config := &Config{
		Host:      "localhost:8080",
		BaseURL:   "http://localhost:8080",
		FilePath:  "storage.json",
		DB:        "host=localhost port=5434 user=loader password=1234 dbname=truecode_db sslmode=disable",
		StoreType: "memory",
	}
	return config
}
