package storage

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/serg2014/shortener/internal/logger"
	"go.uber.org/zap"
)

type item struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
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
	data := make(short2orig)
	// TODO если строка будет длинной получим ошибку
	for scanner.Scan() {
		line := scanner.Bytes()
		err := json.Unmarshal(line, &item)
		if err != nil {
			return nil, err
		}
		if _, ok := data[item.ShortURL]; ok {
			logger.Log.Error(
				"Duplicate key ShortURL",
				zap.String("ShortURL", item.ShortURL),
				zap.String("OriginalURL", item.OriginalURL),
			)
		}
		data[item.ShortURL] = item.OriginalURL
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &storageFile{file: file, storage: storage{short2orig: data}}, nil
}

func (s *storageFile) Set(key string, value string) error {
	s.m.Lock()
	defer s.m.Unlock()
	s.short2orig[key] = value
	err := s.saveRow(key, value)
	if err != nil {
		logger.Log.Error("while save row in file", zap.Error(err))
	}
	return err
}

// TODO flush
func (s *storageFile) saveRow(shortURL, originalURL string) error {
	itemData := item{ShortURL: shortURL, OriginalURL: originalURL}
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

func (storage *storageFile) Close() error {
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
