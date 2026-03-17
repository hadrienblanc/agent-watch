package peer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"claude_monitor/internal/data"
)

func TestFetchPeer(t *testing.T) {
	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/stats" {
			stats := &data.Stats{
				TotalSessions:     5,
				TotalInputTokens:  1000,
				TotalOutputTokens: 500,
				TotalCost:         2.50,
				Models: map[string]int{
					"claude-opus-4-6": 5,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(stats)
		} else if r.URL.Path == "/api/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}
	}))
	defer ts.Close()

	// Test fetch - convert Addr to string
	addr := ts.Listener.Addr().String()
	stats, err := FetchPeer(addr)
	if err != nil {
		t.Fatalf("FetchPeer failed: %v", err)
	}

	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if stats.TotalSessions != 5 {
		t.Errorf("TotalSessions: got %d, want 5", stats.TotalSessions)
	}
	if stats.TotalInputTokens != 1000 {
		t.Errorf("TotalInputTokens: got %d, want 1000", stats.TotalInputTokens)
	}
}

func TestFetchPeerError(t *testing.T) {
	// Test with non-existent server
	_, err := FetchPeer("127.0.0.1:9999")
	if err == nil {
		t.Fatal("Expected error for non-existent server")
	}
}

func TestCheckHealth(t *testing.T) {
	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}
	}))
	defer ts.Close()

	// Test health check - convert Addr to string
	addr := ts.Listener.Addr().String()
	err := CheckHealth(addr)
	if err != nil {
		t.Fatalf("CheckHealth failed: %v", err)
	}
}
