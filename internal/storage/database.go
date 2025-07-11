package storage

import (
	"context"
	"database/sql"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/serg2014/shortener/internal/logger"
	"go.uber.org/zap"
)

const KeyLength = 8

type storageDB struct {
	db *sql.DB
}

func NewStorageDB(dsn string) (Storager, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	query := `CREATE TABLE IF NOT EXISTS short2orig (
		short_url varchar(` + strconv.Itoa(KeyLength) + `) PRIMARY KEY,
		orig_url text
	)`
	_, err = db.ExecContext(context.Background(), query)
	if err != nil {
		return nil, err
	}
	return &storageDB{db: db}, nil
}

// TODO return error
func (storage *storageDB) Get(key string) (string, bool, error) {
	query := "SELECT orig_url FROM short2orig WHERE short_url = $1"
	row := storage.db.QueryRowContext(context.Background(), query, key)
	var value string
	err := row.Scan(&value)
	if err == nil {
		return value, true, nil
	}

	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return "", false, err
}

func (storage *storageDB) Set(key string, value string) error {
	query := `INSERT INTO short2orig (short_url, orig_url)
	 	VALUES ($1, $2)
		ON CONFLICT (short_url) DO NOTHING
	`
	_, err := storage.db.ExecContext(context.Background(), query, key, value)
	if err != nil {
		// TODO может залогировать там где вызов
		logger.Log.Error("can not insert", zap.String("short_url", key), zap.String("orig_url", value))
	}
	return err
}

func (storage *storageDB) Close() error {
	return storage.db.Close()
}
