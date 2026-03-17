package peer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"claude_monitor/internal/data"
)

// PeerStatus represents the status of a peer machine.
type PeerStatus struct {
	Address   string     `json:"address"`
	Online    bool       `json:"online"`
	LastSeen  time.Time  `json:"lastSeen"`
	LastError string     `json:"lastError,omitempty"`
	Stats     *data.Stats `json:"stats,omitempty"`
}

// FetchPeer retrieves stats from a remote peer via HTTP.
func FetchPeer(address string) (*data.Stats, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	url := fmt.Sprintf("http://%s/api/stats", address)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var stats data.Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return &stats, nil
}

// CheckHealth pings a peer to check if it's online.
func CheckHealth(address string) error {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	url := fmt.Sprintf("http://%s/api/health", address)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}
