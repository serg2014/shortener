package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"os"

	"github.com/serg2014/shortener/internal/logger"
	"go.uber.org/zap"
)

type item struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id,omitempty"`
}

type storageFile struct {
	storage
	file *os.File
}

func NewStorageFile(filePath string) (Storager, error) {
	// os.O_APPEND os.O_SYNC
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	var item item
	s := storageFile{file: file, storage: storage{short2orig: make(short2orig), orig2short: make(orig2short)}}
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
		s.users[item.UserID][item.ShortURL] = item.OriginalURL
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &s, nil
}

/*
func (s *storageFile) Set(ctx context.Context, key string, value string, userID string) error {
	s.m.Lock()
	defer s.m.Unlock()
	if _, ok := s.orig2short[value]; ok {
		return ErrConflict
	}

	s.short2orig[key] = value
	s.orig2short[value] = key
	s.users[userID][key] = value

	err := s.saveRow(key, value, userID)
	if err != nil {
		logger.Log.Error("while save row in file", zap.Error(err))
	}
	return err
}
*/

func (s *storageFile) SetBatch(ctx context.Context, data short2orig, userID string) error {
	s.m.Lock()
	defer s.m.Unlock()

	for key, value := range data {
		s.short2orig[key] = value
		// TODO проблема если в data несколько одинаковых значений
		s.orig2short[value] = key
		s.users[userID][key] = value
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
	itemData := item{ShortURL: shortURL, OriginalURL: originalURL, UserID: userID}
	line, err := json.Marshal(itemData)
	if err != nil {
		return err
	}
	_, err = s.file.Write(line)
	if err != nil {
		return err
	}
	_, err = s.file.WriteString("\n")
	if err != nil {
		return err
	}
	return nil
}

func (s *storageFile) Close() error {
	// TODO flush
	return nil
}

/*
func (s *storageFile) SaveAllData() error {
	s.m.Lock()
	defer s.m.Unlock()
	s.file.Seek(0, io.SeekStart)

	for k := range s.short2orig {
		// TODO в случае ошибки не запишем хвост
		err := s.saveRow(k, s.short2orig[k])
		if err != nil {
			return err
		}
	}
	return nil
}
*/
