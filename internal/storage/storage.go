package storage

import (
	"context"
	"maps"
	"sync"

	"errors"

	"github.com/serg2014/shortener/internal/logger"
	"go.uber.org/zap"
)

type short2orig map[string]string
type orig2short map[string]string
type users map[string]short2orig
type storage struct {
	// TODO неоптимально по памяти
	users      users
	short2orig short2orig
	orig2short orig2short
	m          sync.RWMutex
}

func NewStorage(ctx context.Context, filePath string, dsn string) (Storager, error) {
	if dsn != "" {
		return NewStorageDB(ctx, dsn)
	}
	if filePath != "" {
		return NewStorageFile(filePath)
	}
	return NewStorageMemory()
}

func NewStorageMemory() (Storager, error) {
	return &storage{
			short2orig: make(short2orig),
			orig2short: make(orig2short),
			users:      make(users),
		},
		nil
}

func (s *storage) Get(ctx context.Context, key string) (string, bool, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	v, ok := s.short2orig[key]
	return v, ok, nil
}

func (s *storage) GetUserURLS(ctx context.Context, userID string) ([]item, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	v, ok := s.users[userID]
	if !ok {
		return nil, errors.New("no data for user_id")
	}
	result := make([]item, 0, len(v))
	for short, url := range v {
		result = append(result, item{ShortURL: short, OriginalURL: url})
	}
	return result, nil
}

func (s *storage) GetShort(ctx context.Context, origURL string) (string, bool, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	v, ok := s.orig2short[origURL]
	return v, ok, nil
}

func (s *storage) Set(ctx context.Context, key string, value string, userID string) error {
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

func (s *storage) saveRow(shortURL, originalURL string, userID string) error {
	return nil
}

func (s *storage) SetBatch(ctx context.Context, data short2orig, userID string) error {
	s.m.Lock()
	defer s.m.Unlock()

	maps.Copy(s.short2orig, data)
	for k := range data {
		// TODO проблема если в data несколько одинаковых значений
		s.orig2short[data[k]] = k
		s.users[userID][k] = data[k]
	}
	return nil
}

func (s *storage) Close() error {
	return nil
}

func (s *storage) Ping(ctx context.Context) error {
	return nil
}

type Storager interface {
	Get(ctx context.Context, key string) (string, bool, error)
	GetUserURLS(ctx context.Context, userID string) ([]item, error)
	GetShort(ctx context.Context, origURL string) (string, bool, error)
	Set(ctx context.Context, key string, value string, userID string) error
	SetBatch(ctx context.Context, data short2orig, userID string) error
	Close() error
	Ping(ctx context.Context) error
}
