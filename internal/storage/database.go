package storage

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const KeyLength = 8

var ErrConflict = errors.New("data conflict")

type storageDB struct {
	db *sql.DB
}

func NewStorageDB(ctx context.Context, dsn string) (Storager, error) {
	// dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
	//  `localhost`, `video`, `XXXXXXXX`, `video`)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	query := `CREATE TABLE IF NOT EXISTS short2orig (
		short_url varchar(` + strconv.Itoa(KeyLength) + `) PRIMARY KEY,
		orig_url text UNIQUE
	)`
	_, err = db.ExecContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &storageDB{db: db}, nil
}

func (storage *storageDB) Get(ctx context.Context, key string) (string, bool, error) {
	query := "SELECT orig_url FROM short2orig WHERE short_url = $1"
	row := storage.db.QueryRowContext(ctx, query, key)
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

func (storage *storageDB) GetShort(ctx context.Context, url string) (string, bool, error) {
	query := "SELECT short_url FROM short2orig WHERE orig_url = $1"
	row := storage.db.QueryRowContext(ctx, query, url)
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

func (storage *storageDB) Set(ctx context.Context, key string, value string) error {
	query := `INSERT INTO short2orig (short_url, orig_url)
	 	VALUES ($1, $2)
		ON CONFLICT (orig_url) DO NOTHING
	`
	result, err := storage.db.ExecContext(ctx, query, key, value)
	if err != nil {
		return err
	}
	ra, _ := result.RowsAffected()
	if ra == 0 {
		return ErrConflict
	}

	return err
}

func (storage *storageDB) SetBatch(ctx context.Context, data short2orig) error {
	// начать транзакцию
	tx, err := storage.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO short2orig (short_url, orig_url)
		VALUES ($1, $2)
		ON CONFLICT (orig_url) DO NOTHING
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for key, value := range data {
		_, err := stmt.ExecContext(ctx, key, value)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (storage *storageDB) Close() error {
	return storage.db.Close()
}

func (storage *storageDB) Ping(ctx context.Context) error {
	return storage.db.PingContext(ctx)
}
