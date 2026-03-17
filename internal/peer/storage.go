package peer

import "sync"

// Storage manages in-memory peer addresses.
type Storage struct {
	mu    sync.RWMutex
	peers []string
}

// NewStorage creates a new in-memory peer storage.
func NewStorage() *Storage {
	return &Storage{
		peers: []string{},
	}
}

// List returns all stored peer addresses.
func (s *Storage) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, len(s.peers))
	copy(result, s.peers)
	return result
}

// Add stores a new peer address.
func (s *Storage) Add(address string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.peers {
		if p == address {
			return
		}
	}

	s.peers = append(s.peers, address)
}

// Remove deletes a peer address.
func (s *Storage) Remove(address string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, p := range s.peers {
		if p == address {
			s.peers = append(s.peers[:i], s.peers[i+1:]...)
			return
		}
	}
}
