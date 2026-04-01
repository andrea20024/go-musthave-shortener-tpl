package config

type Config struct {
	Host     string `env:"SERVER_ADDRESS"`
	BaseURL  string `env:"BASE_URL"`
	FilePath string `env:"FILE_STORAGE_PATH"`
}

func InitConfig() *Config {
	config := &Config{
		Host:     "localhost:8080",
		BaseURL:  "http://localhost:8080",
		FilePath: "storage.json",
	}
	return config
}
