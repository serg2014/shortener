package storage

import (
	"sync"
)

type short2orig map[string]string
type storage struct {
	short2orig short2orig
	m          sync.RWMutex
}

func NewStorage(filePath string, dsn string) (Storager, error) {
	if dsn != "" {
		return NewStorageDB(dsn)
	}
	if filePath != "" {
		return NewStorageFile(filePath)
	}
	return NewStorageMemory(nil)
}

func NewStorageMemory(data map[string]string) (Storager, error) {
	if data == nil {
		return &storage{short2orig: make(map[string]string)}, nil
	}
	return &storage{short2orig: data}, nil
}

func (s *storage) Get(key string) (string, bool, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	v, ok := s.short2orig[key]
	return v, ok, nil
}

func (s *storage) Set(key string, value string) error {
	s.m.Lock()
	defer s.m.Unlock()
	s.short2orig[key] = value
	return nil
}

func (s *storage) Close() error {
	return nil
}

type Storager interface {
	Get(key string) (string, bool, error)
	Set(key string, value string) error
	Close() error
}
