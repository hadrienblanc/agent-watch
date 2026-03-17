package peer

import (
	"sync"
	"testing"
)

func TestNewStorage(t *testing.T) {
	s := NewStorage()
	if s == nil {
		t.Fatal("NewStorage returned nil")
	}
	if len(s.List()) != 0 {
		t.Errorf("new storage should be empty, got %d peers", len(s.List()))
	}
}

func TestStorageAddAndList(t *testing.T) {
	s := NewStorage()
	s.Add("192.168.1.10:9999")
	s.Add("192.168.1.20:9999")

	peers := s.List()
	if len(peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(peers))
	}
	if peers[0] != "192.168.1.10:9999" {
		t.Errorf("peers[0] = %q, want %q", peers[0], "192.168.1.10:9999")
	}
	if peers[1] != "192.168.1.20:9999" {
		t.Errorf("peers[1] = %q, want %q", peers[1], "192.168.1.20:9999")
	}
}

func TestStorageAddDuplicate(t *testing.T) {
	s := NewStorage()
	s.Add("192.168.1.10:9999")
	s.Add("192.168.1.10:9999")
	s.Add("192.168.1.10:9999")

	peers := s.List()
	if len(peers) != 1 {
		t.Errorf("expected 1 peer after duplicates, got %d", len(peers))
	}
}

func TestStorageRemove(t *testing.T) {
	s := NewStorage()
	s.Add("192.168.1.10:9999")
	s.Add("192.168.1.20:9999")
	s.Add("192.168.1.30:9999")

	s.Remove("192.168.1.20:9999")

	peers := s.List()
	if len(peers) != 2 {
		t.Fatalf("expected 2 peers after remove, got %d", len(peers))
	}
	for _, p := range peers {
		if p == "192.168.1.20:9999" {
			t.Error("removed peer should not be in list")
		}
	}
}

func TestStorageRemoveNonExistent(t *testing.T) {
	s := NewStorage()
	s.Add("192.168.1.10:9999")

	s.Remove("192.168.1.99:9999") // does not exist

	peers := s.List()
	if len(peers) != 1 {
		t.Errorf("removing non-existent peer should not change list, got %d peers", len(peers))
	}
}

func TestStorageRemoveFromEmpty(t *testing.T) {
	s := NewStorage()
	s.Remove("192.168.1.10:9999") // no panic

	if len(s.List()) != 0 {
		t.Error("empty storage should remain empty after remove")
	}
}

func TestStorageListReturnsCopy(t *testing.T) {
	s := NewStorage()
	s.Add("192.168.1.10:9999")

	list := s.List()
	list[0] = "mutated"

	// Original should be unchanged
	peers := s.List()
	if peers[0] != "192.168.1.10:9999" {
		t.Errorf("List should return a copy, but original was mutated to %q", peers[0])
	}
}

func TestStorageConcurrency(t *testing.T) {
	s := NewStorage()
	var wg sync.WaitGroup

	// Concurrent adds
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			addr := "192.168.1." + string(rune('0'+i%10)) + ":9999"
			s.Add(addr)
		}(i)
	}

	// Concurrent reads
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.List()
		}()
	}

	// Concurrent removes
	for i := range 10 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			addr := "192.168.1." + string(rune('0'+i%10)) + ":9999"
			s.Remove(addr)
		}(i)
	}

	wg.Wait()
	// No race conditions or panics = pass
}
