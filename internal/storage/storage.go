package storage

import (
	"context"
	"maps"
	"sync"
)

type short2orig map[string]string
type orig2short map[string]string
type storage struct {
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
	return NewStorageMemory(nil)
}

func NewStorageMemory(data map[string]string) (Storager, error) {
	if data == nil {
		return &storage{short2orig: make(short2orig), orig2short: make(orig2short)}, nil
	}
	orig2short := make(orig2short, len(data))
	for k := range data {
		orig2short[data[k]] = k
	}
	return &storage{short2orig: data, orig2short: orig2short}, nil
}

func (s *storage) Get(ctx context.Context, key string) (string, bool, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	v, ok := s.short2orig[key]
	return v, ok, nil
}

func (s *storage) GetShort(ctx context.Context, origURL string) (string, bool, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	v, ok := s.orig2short[origURL]
	return v, ok, nil
}

func (s *storage) Set(ctx context.Context, key string, value string) error {
	s.m.Lock()
	defer s.m.Unlock()
	if _, ok := s.orig2short[value]; ok {
		return ErrConflict
	}

	s.short2orig[key] = value
	s.orig2short[value] = key
	return nil
}

func (s *storage) SetBatch(ctx context.Context, data short2orig) error {
	s.m.Lock()
	defer s.m.Unlock()

	maps.Copy(s.short2orig, data)
	for k := range data {
		// TODO проблема если в data несколько одинаковых значений
		s.orig2short[data[k]] = k
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
	GetShort(ctx context.Context, origURL string) (string, bool, error)
	Set(ctx context.Context, key string, value string) error
	SetBatch(ctx context.Context, data short2orig) error
	Close() error
	Ping(ctx context.Context) error
}
