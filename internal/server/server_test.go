package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"claude_monitor/internal/data"
)

func TestServerEndpoints(t *testing.T) {
	stats := &data.Stats{
		TotalSessions:     10,
		TotalInputTokens:  5000,
		TotalOutputTokens: 2000,
		TotalCost:          5.50,
	}

	// Create server with test stats provider
	srv := New(9999, func() interface{} { return stats })

	handler := http.NewServeMux()
	handler.HandleFunc("/api/stats", srv.HandleStats)
	handler.HandleFunc("/api/health", srv.HandleHealth)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Test /api/health
	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("Health request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	var healthResult map[string]string
	json.NewDecoder(resp.Body).Decode(&healthResult)
	if healthResult["status"] != "ok" {
		t.Errorf("Expected status=ok, got %s", healthResult["status"])
	}
	resp.Body.Close()

	// Test /api/stats
	resp, err = http.Get(ts.URL + "/api/stats")
	if err != nil {
		t.Fatalf("Stats request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	var statsResult data.Stats
	json.NewDecoder(resp.Body).Decode(&statsResult)
	if statsResult.TotalSessions != 10 {
		t.Errorf("Expected TotalSessions=10, got %d", statsResult.TotalSessions)
	}
	if statsResult.TotalCost != 5.50 {
		t.Errorf("Expected TotalCost=5.50, got %.2f", statsResult.TotalCost)
	}
	resp.Body.Close()
}

func TestServerStatsNil(t *testing.T) {
	// Create server with nil provider
	srv := New(9999, func() interface{} { return nil })

	handler := http.NewServeMux()
	handler.HandleFunc("/api/stats", srv.HandleStats)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/stats")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
