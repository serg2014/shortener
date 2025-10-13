package storage

import (
	"context"
	"maps"
	"sync"

	"github.com/serg2014/shortener/internal/models"
)

type Short2orig map[string]string
type orig2short map[string]string
type users map[string]Short2orig
type storage struct {
	// TODO неоптимально по памяти
	users      users
	short2orig Short2orig
	orig2short orig2short
	m          sync.RWMutex
}

type Message struct {
	UserID   string
	ShortURL []string
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
			short2orig: make(Short2orig),
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

func (s *storage) GetUserURLS(ctx context.Context, userID string) ([]Item, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	v, ok := s.users[userID]
	if !ok {
		return make([]Item, 0), nil
	}
	result := make([]Item, 0, len(v))
	for short, url := range v {
		result = append(result, Item{ShortURL: short, OriginalURL: url})
	}
	return result, nil
}

func (s *storage) GetShort(ctx context.Context, origURL string) (string, bool, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	v, ok := s.orig2short[origURL]
	return v, ok, nil
}

func (s *storage) set(ctx context.Context, key string, value string, userID string) error {
	if _, ok := s.orig2short[value]; ok {
		return ErrConflict
	}

	s.short2orig[key] = value
	s.orig2short[value] = key
	if _, ok := s.users[userID]; !ok {
		s.users[userID] = make(Short2orig)
	}
	s.users[userID][key] = value
	return nil
}

func (s *storage) Set(ctx context.Context, key string, value string, userID string) error {
	s.m.Lock()
	defer s.m.Unlock()
	return s.set(ctx, key, value, userID)
}

func (s *storage) SetBatch(ctx context.Context, data Short2orig, userID string) error {
	s.m.Lock()
	defer s.m.Unlock()

	if _, ok := s.users[userID]; !ok {
		s.users[userID] = make(Short2orig)
	}

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

func (s *storage) DeleteUserURLS(ctx context.Context, req models.RequestForDeleteURLS, userID string) error {
	// TODO
	return nil
}

type Storager interface {
	Get(ctx context.Context, key string) (string, bool, error)
	GetUserURLS(ctx context.Context, userID string) ([]Item, error)
	GetShort(ctx context.Context, origURL string) (string, bool, error)
	Set(ctx context.Context, key string, value string, userID string) error
	SetBatch(ctx context.Context, data Short2orig, userID string) error
	Close() error
	Ping(ctx context.Context) error
	DeleteUserURLS(ctx context.Context, req models.RequestForDeleteURLS, userID string) error
}
