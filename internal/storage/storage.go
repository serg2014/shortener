package storage

import "sync"

type storage struct {
	short2orig map[string]string
	m          sync.RWMutex
}

func NewStorage(data map[string]string) *storage {
	if data == nil {
		return &storage{short2orig: make(map[string]string)}
	}
	return &storage{short2orig: data}
}

func (s *storage) Get(key string) (string, bool) {
	s.m.RLock()
	defer s.m.RUnlock()
	v, ok := s.short2orig[key]
	return v, ok
}

func (s *storage) Set(key string, value string) {
	s.m.Lock()
	defer s.m.Unlock()
	s.short2orig[key] = value
}

type Storager interface {
	Get(key string) (string, bool)
	Set(key string, value string)
}
