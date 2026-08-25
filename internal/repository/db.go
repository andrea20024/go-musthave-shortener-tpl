package storage

// Package storage provides PostgreSQL implementation of the Repository interface.
//
// This file contains the dbRepository struct which implements all methods of
// the Repository interface using PostgreSQL via the database/sql and pgx drivers.
// The Init function handles database connection and automatic schema migration.

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

// dbRepository implements the Repository interface using PostgreSQL.
type dbRepository struct {
	db *sql.DB
}

// Add stores a URL mapping in PostgreSQL, returning DuplicateError on conflict.
func (r *dbRepository) Add(key string, url string, userID string) error {
	result, err := r.db.Exec("INSERT INTO urls (short_url, original_url, user_id) VALUES ($1, $2, $3) ON CONFLICT ON CONSTRAINT urls_original_url_key DO NOTHING", key, url, userID)
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

// Get retrieves the original URL and deletion status by short URL key.
func (r *dbRepository) Get(key string) (string, error) {
	var url string
	var isDeleted bool
	err := r.db.QueryRow("SELECT original_url, is_deleted FROM urls WHERE short_url = $1", key).Scan(&url, &isDeleted)
	if err != nil {
		return "", err
	}
	if isDeleted {
		return "", &DeletedError{}
	}
	return url, nil
}

// GetKeyByURL finds the short URL key for a given original URL.
func (r *dbRepository) GetKeyByURL(url string) (string, error) {
	var key string
	err := r.db.QueryRow("SELECT short_url FROM urls WHERE original_url = $1 AND is_deleted = false", url).Scan(&key)
	return key, err
}

// GetUserURLs retrieves all non-deleted URLs shortened by a specific user.
func (r *dbRepository) GetUserURLs(userID string) ([]UserURL, error) {
	rows, err := r.db.Query("SELECT short_url, original_url FROM urls WHERE user_id = $1 AND is_deleted = false", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []UserURL
	for rows.Next() {
		var shortURL, originalURL string
		if err := rows.Scan(&shortURL, &originalURL); err != nil {
			return nil, err
		}
		urls = append(urls, UserURL{
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}

// Ping checks if the PostgreSQL database is reachable.
func (r *dbRepository) Ping() error {
	return r.db.Ping()
}

// DeleteUserURLs marks the specified URLs as deleted for the given user.
func (r *dbRepository) DeleteUserURLs(userID string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	_, err := r.db.Exec(
		"UPDATE urls SET is_deleted = TRUE WHERE user_id = $1 AND short_url = ANY($2) AND is_deleted = FALSE",
		userID, keys,
	)
	return err
}

// AddBatch stores multiple URL mappings in a single transaction.
func (r *dbRepository) AddBatch(urls map[string]string, userID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare("INSERT INTO urls (short_url, original_url, user_id) VALUES ($1, $2, $3) ON CONFLICT ON CONSTRAINT urls_original_url_key DO NOTHING")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for key, url := range urls {
		result, err := stmt.Exec(key, url, userID)
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

// IsDuplicateError reports whether err is a PostgreSQL unique violation.
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

// IsDeletedError reports whether err is a DeletedError.
func (r *dbRepository) IsDeletedError(err error) bool {
	var delErr *DeletedError
	return errors.As(err, &delErr)
}

// Shutdown gracefully closes the database repository.
func (r *dbRepository) Shutdown() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// Init connects to PostgreSQL, runs migrations, and returns a Repository.
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
