package peer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hadrienblanc/agent-watch/internal/data"
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

func TestCheckHealthError(t *testing.T) {
	err := CheckHealth("127.0.0.1:1") // port 1 unlikely to be listening
	if err == nil {
		t.Fatal("Expected error for unreachable health endpoint")
	}
}

func TestCheckHealthNon200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	err := CheckHealth(ts.Listener.Addr().String())
	if err == nil {
		t.Fatal("Expected error for non-200 health response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status code, got: %v", err)
	}
}

func TestFetchPeerNon200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("stats not available"))
	}))
	defer ts.Close()

	_, err := FetchPeer(ts.Listener.Addr().String())
	if err == nil {
		t.Fatal("Expected error for 503 response")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("error should mention status code, got: %v", err)
	}
}

func TestFetchPeerInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{not json"))
	}))
	defer ts.Close()

	_, err := FetchPeer(ts.Listener.Addr().String())
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("error should mention 'invalid JSON', got: %v", err)
	}
}

func TestFetchPeerFullStats(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats := &data.Stats{
			TotalSessions:     10,
			TotalInputTokens:  50000,
			TotalOutputTokens: 20000,
			TotalCacheRead:    10000,
			TotalCost:         5.75,
			TotalMessages:     100,
			TotalToolUses:     30,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}))
	defer ts.Close()

	stats, err := FetchPeer(ts.Listener.Addr().String())
	if err != nil {
		t.Fatalf("FetchPeer failed: %v", err)
	}

	if stats.TotalSessions != 10 {
		t.Errorf("TotalSessions = %d, want 10", stats.TotalSessions)
	}
	if stats.TotalCacheRead != 10000 {
		t.Errorf("TotalCacheRead = %d, want 10000", stats.TotalCacheRead)
	}
	if stats.TotalCost != 5.75 {
		t.Errorf("TotalCost = %.2f, want 5.75", stats.TotalCost)
	}
	if stats.TotalMessages != 100 {
		t.Errorf("TotalMessages = %d, want 100", stats.TotalMessages)
	}
}
