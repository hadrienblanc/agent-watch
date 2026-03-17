package peer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Storage manages persistent peer addresses.
type Storage struct {
	filePath string
	mu       sync.RWMutex
	peers    []string
}

// NewStorage creates a new peer storage.
func NewStorage() (*Storage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(home, ".claude_monitor")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	filePath := filepath.Join(dir, "peers.json")

	s := &Storage{
		filePath: filePath,
		peers:    []string{},
	}

	if err := s.load(); err != nil {
		// File doesn't exist yet, that's OK
		return s, nil
	}

	return s, nil
}

// load reads peers from the JSON file.
func (s *Storage) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.peers)
}

// save writes peers to the JSON file.
func (s *Storage) save() error {
	data, err := json.MarshalIndent(s.peers, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
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
func (s *Storage) Add(address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already exists
	for _, p := range s.peers {
		if p == address {
			return nil
		}
	}

	s.peers = append(s.peers, address)
	return s.save()
}

// Remove deletes a peer address.
func (s *Storage) Remove(address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, p := range s.peers {
		if p == address {
			s.peers = append(s.peers[:i], s.peers[i+1:]...)
			return s.save()
		}
	}

	return nil
}
