package storage

import (
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type dbRepository struct {
	db *sql.DB
}

func (r *dbRepository) Add(key string, url string) error {
	result, err := r.db.Exec("INSERT INTO urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT ON CONSTRAINT urls_original_url_key DO NOTHING", key, url)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return &pgconn.PgError{Code: pgerrcode.UniqueViolation}
	}
	return nil
}

func (r *dbRepository) Get(key string) (string, error) {
	var url string
	err := r.db.QueryRow("SELECT original_url FROM urls WHERE short_url = $1", key).Scan(&url)
	return url, err
}

func (r *dbRepository) GetKeyByURL(url string) (string, error) {
	var key string
	err := r.db.QueryRow("SELECT short_url FROM urls WHERE original_url = $1", url).Scan(&key)
	return key, err
}

func (r *dbRepository) Ping() error {
	return r.db.Ping()
}

func (r *dbRepository) AddBatch(urls map[string]string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare("INSERT INTO urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT ON CONSTRAINT urls_original_url_key DO NOTHING")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for key, url := range urls {
		result, err := stmt.Exec(key, url)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return &pgconn.PgError{Code: pgerrcode.UniqueViolation}
		}
	}

	return tx.Commit()
}

func (r *dbRepository) IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == pgerrcode.UniqueViolation
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

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		db.Close()
		return nil
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		db.Close()
		return nil
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		db.Close()
		return nil
	}

	return &dbRepository{db: db}
}
