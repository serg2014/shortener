package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/serg2014/shortener/internal/logger"
	"github.com/serg2014/shortener/internal/models"
	"go.uber.org/zap"
)

// размер столбца short_url в таблице short2orig
const KeyLength = 8

var ErrConflict = errors.New("data conflict")
var ErrDeleted = errors.New("data deleted")

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
	// проверяем подключение к бд
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %v", err)
	}
	logger.Log.Info("Connected to db")

	// миграции
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, err
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://../../migrations",
		dsn,
		driver,
	)
	if err != nil {
		return nil, err
	}
	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		//logger.Log.Fatal("failed to apply migrations", zap.Error(err))
		return nil, err
	}

	return &storageDB{db: db}, nil
}

func (storage *storageDB) Get(ctx context.Context, key string) (string, bool, error) {
	query := "SELECT orig_url, is_deleted FROM short2orig WHERE short_url = $1"
	row := storage.db.QueryRowContext(ctx, query, key)
	var value string
	var deleted bool
	err := row.Scan(&value, &deleted)
	if err == nil {
		if deleted {
			return "", false, ErrDeleted
		}
		return value, true, nil
	}

	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return "", false, err
}

func (storage *storageDB) GetUserURLS(ctx context.Context, userID string) ([]item, error) {
	query := "SELECT short_url, orig_url FROM short2orig WHERE user_id = $1"
	rows, err := storage.db.QueryContext(ctx, query, userID)
	if err != nil {
		logger.Log.Info("select user urls", zap.Error(err))
		return nil, err
	}
	// обязательно закрываем перед возвратом функции
	defer rows.Close()

	result := make([]item, 0, 1)
	// пробегаем по всем записям
	for rows.Next() {
		var item item
		err = rows.Scan(&item.ShortURL, &item.OriginalURL)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return result, nil
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

func (storage *storageDB) Set(ctx context.Context, key string, value string, userID string) error {
	query := `INSERT INTO short2orig (short_url, orig_url, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (orig_url) DO NOTHING
	`
	result, err := storage.db.ExecContext(ctx, query, key, value, userID)
	if err != nil {
		return err
	}
	ra, _ := result.RowsAffected()
	if ra == 0 {
		return ErrConflict
	}

	return err
}

func (storage *storageDB) SetBatch(ctx context.Context, data short2orig, userID string) error {
	// начать транзакцию
	tx, err := storage.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO short2orig (short_url, orig_url, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (orig_url) DO NOTHING
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for key, value := range data {
		_, err := stmt.ExecContext(ctx, key, value, userID)
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

func (storage *storageDB) DeleteUserURLS(ctx context.Context, data models.RequestForDeleteURLS, userID string) error {
	// начать транзакцию
	tx, err := storage.db.Begin()
	if err != nil {
		logger.Log.Error("begin", zap.Error(err))
		return err
	}
	defer tx.Rollback()

	query := `UPDATE short2orig SET is_deleted=true
		WHERE short_url=$1 and user_id=$2
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		logger.Log.Error("prepare", zap.Error(err))
		return err
	}
	defer stmt.Close()

	for i := range data {
		_, err := stmt.ExecContext(ctx, data[i], userID)
		if err != nil {
			logger.Log.Error("update", zap.Error(err))
		}

	}

	return tx.Commit()
}
