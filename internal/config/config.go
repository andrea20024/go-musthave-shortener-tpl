package config

type Config struct {
	Host     string `env:"SERVER_ADDRESS"`
	BaseURL  string `env:"BASE_URL"`
	FilePath string `env:"FILE_STORAGE_PATH"`
	Db       string `env:"DATABASE_DSN"`
}

func InitConfig() *Config {
	config := &Config{
		Host:     "localhost:8080",
		BaseURL:  "http://localhost:8080",
		FilePath: "storage.json",
		Db:       "host=localhost port=5434 user=loader password=1234 dbname=truecode_db sslmode=disable",
	}
	return config
}
