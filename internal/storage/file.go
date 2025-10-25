package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"

	"go.uber.org/zap"

	"github.com/serg2014/shortener/internal/logger"
)

// Item one row file representation
type Item struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id,omitempty"`
}

type storageFile struct {
	storage
	file io.ReadWriter
}

func newStorageIO(file io.ReadWriter) (Storager, error) {
	scanner := bufio.NewScanner(file)
	var item Item
	s := storageFile{
		file: file,
		storage: storage{
			short2orig: make(Short2orig),
			orig2short: make(orig2short),
			users:      make(users),
		},
	}

	// TODO если строка будет длинной получим ошибку
	for scanner.Scan() {
		line := scanner.Bytes()
		err := json.Unmarshal(line, &item)
		if err != nil {
			return nil, err
		}
		if _, ok := s.short2orig[item.ShortURL]; ok {
			logger.Log.Error(
				"Duplicate key ShortURL",
				zap.String("ShortURL", item.ShortURL),
				zap.String("OriginalURL", item.OriginalURL),
			)
			continue
		}
		if _, ok := s.orig2short[item.OriginalURL]; ok {
			logger.Log.Error(
				"Duplicate key OriginalURL",
				zap.String("ShortURL", item.ShortURL),
				zap.String("OriginalURL", item.OriginalURL),
			)
			continue
		}
		s.short2orig[item.ShortURL] = item.OriginalURL
		s.orig2short[item.OriginalURL] = item.ShortURL

		if _, ok := s.users[item.UserID]; !ok {
			s.users[item.UserID] = make(Short2orig)
		}
		s.users[item.UserID][item.ShortURL] = item.OriginalURL
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &s, nil
}

// NewStorageFile create file storage type *storageFile
func NewStorageFile(filePath string) (Storager, error) {
	// os.O_APPEND os.O_SYNC
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return newStorageIO(file)
}

// Set save record in file
func (s *storageFile) Set(ctx context.Context, key string, value string, userID string) error {
	s.m.Lock()
	defer s.m.Unlock()
	err := s.set(key, value, userID)
	if err != nil {
		return err
	}

	return s.saveRow(key, value, userID)
}

// SetBatch save range of data into file
// BUG(Serg): если в data есть повторяющиеся данные запишем несколько строк вместо одной
func (s *storageFile) SetBatch(ctx context.Context, data Short2orig, userID string) error {
	s.m.Lock()
	defer s.m.Unlock()

	if _, ok := s.users[userID]; !ok {
		s.users[userID] = make(Short2orig)
	}

	for key, value := range data {
		s.short2orig[key] = value
		// TODO проблема если в data несколько одинаковых значений
		s.orig2short[value] = key
		s.users[userID][key] = value
		// TODO писать надо чанками
		err := s.saveRow(key, value, userID)
		if err != nil {
			logger.Log.Error("while save row in file", zap.Error(err))
			return err
		}
	}
	return nil
}

// TODO flush
func (s *storageFile) saveRow(shortURL, originalURL string, userID string) error {
	itemData := Item{ShortURL: shortURL, OriginalURL: originalURL, UserID: userID}
	line, err := json.Marshal(itemData)
	if err != nil {
		return err
	}
	_, err = s.file.Write(line)
	if err != nil {
		return err
	}
	_, err = s.file.Write([]byte("\n"))
	if err != nil {
		return err
	}
	return nil
}

// Close close connect to file/db
func (s *storageFile) Close() error {
	// TODO flush
	return nil
}
