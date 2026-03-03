package config

type Config struct {
	Host    string
	BaseURL string
}

func InitConfig() *Config {
	config := &Config{
		Host:    "localhost:8080",
		BaseURL: "http://localhost:8080",
	}
	return config
}
