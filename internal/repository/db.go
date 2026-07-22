package storage

import (
	"database/sql"

	"github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var conf *config.Config

func Init(cfg *config.Config) {
	conf = cfg
}

func Ping() error {
	db, err := sql.Open("pgx", conf.Db)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return err
	}

	return nil
}
