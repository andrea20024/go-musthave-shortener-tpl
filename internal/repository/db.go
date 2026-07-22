package storage

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type dbRepository struct {
	db *sql.DB
}

func (r *dbRepository) Add(key string, url string) {
	r.db.Exec("INSERT INTO urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT (short_url) DO NOTHING", key, url)
}

func (r *dbRepository) Get(key string) string {
	var url string
	r.db.QueryRow("SELECT original_url FROM urls WHERE short_url = $1", key).Scan(&url)
	return url
}

func (r *dbRepository) Ping() error {
	return r.db.Ping()
}

func Init(conn string) Repository {
	db, err := sql.Open("pgx", conn)
	if err != nil {
		return nil
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		short_url VARCHAR(255) UNIQUE NOT NULL,
		original_url TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`)
	return &dbRepository{db: db}
}
