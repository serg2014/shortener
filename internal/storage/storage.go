package storage

type Storage struct {
	short2orig map[string]string
}

func NewStorage(data map[string]string) *Storage {
	if data == nil {
		return &Storage{short2orig: make(map[string]string)}
	}
	return &Storage{short2orig: data}
}

func (s *Storage) Get(key string) (string, bool) {
	v, ok := s.short2orig[key]
	return v, ok
}

func (s *Storage) Set(key string, value string) {
	s.short2orig[key] = value
}
