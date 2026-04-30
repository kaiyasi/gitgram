package store

import "sync"

type DeliveryStore interface {
	Claim(id string) bool
	Release(id string)
}

type MemoryDeliveryStore struct {
	mu    sync.Mutex
	max   int
	seen  map[string]struct{}
	order []string
}

func NewMemoryDeliveryStore(max int) *MemoryDeliveryStore {
	if max <= 0 {
		max = 10000
	}
	return &MemoryDeliveryStore{
		max:  max,
		seen: make(map[string]struct{}),
	}
}

func (s *MemoryDeliveryStore) Claim(id string) bool {
	if id == "" {
		return true
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.seen[id]; ok {
		return false
	}

	s.seen[id] = struct{}{}
	s.order = append(s.order, id)
	s.trimLocked()
	return true
}

func (s *MemoryDeliveryStore) Release(id string) {
	if id == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.seen, id)
	for i, existing := range s.order {
		if existing == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			return
		}
	}
}

func (s *MemoryDeliveryStore) trimLocked() {
	for len(s.order) > s.max {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.seen, oldest)
	}
}
