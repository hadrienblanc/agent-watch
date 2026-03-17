package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDecodeProjectName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"-home-hadrienblanc-Projets-tests-form-on-terminal", "form-on-terminal"},
		{"-home-hadrienblanc-Projets-hadrienblanc-phira", "hadrienblanc-phira"},
	}
	for _, tt := range tests {
		result := decodeProjectName(tt.input)
		if result != tt.expected {
			t.Errorf("decodeProjectName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestParseSession(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test-session.jsonl")

	entries := []Entry{
		{
			Type:      "user",
			Timestamp: time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC),
			SessionID: "sess-123",
			Slug:      "test-convo",
			Version:   "2.1.72",
			Message:   &Message{Role: "user"},
		},
		{
			Type:      "assistant",
			Timestamp: time.Date(2026, 3, 17, 10, 0, 5, 0, time.UTC),
			SessionID: "sess-123",
			DurationMs: 2500,
			Message: &Message{
				Role:  "assistant",
				Model: "claude-opus-4-6",
				Usage: &Usage{
					InputTokens:          5000,
					OutputTokens:         2000,
					CacheReadInputTokens: 3000,
				},
				Content: []Content{
					{Type: "tool_use", Name: "Read"},
					{Type: "tool_use", Name: "Bash"},
					{Type: "text", Text: "Done"},
				},
			},
		},
		{
			Type:      "assistant",
			Timestamp: time.Date(2026, 3, 17, 10, 1, 0, 0, time.UTC),
			SessionID: "sess-123",
			DurationMs: 1500,
			Message: &Message{
				Role:  "assistant",
				Model: "claude-opus-4-6",
				Usage: &Usage{InputTokens: 3000, OutputTokens: 1000},
				Content: []Content{
					{Type: "tool_use", Name: "Read"},
					{Type: "tool_result", IsError: true},
				},
			},
		},
	}

	file, err := os.Create(f)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(file)
	for _, e := range entries {
		enc.Encode(e)
	}
	file.Close()

	session, err := parseSession(f)
	if err != nil {
		t.Fatal(err)
	}

	if session.Slug != "test-convo" {
		t.Errorf("slug = %q, want %q", session.Slug, "test-convo")
	}
	if session.UserMessages != 1 {
		t.Errorf("user messages = %d, want 1", session.UserMessages)
	}
	if session.AssistantMessages != 2 {
		t.Errorf("assistant messages = %d, want 2", session.AssistantMessages)
	}
	if session.InputTokens != 8000 {
		t.Errorf("input tokens = %d, want 8000", session.InputTokens)
	}
	if session.OutputTokens != 3000 {
		t.Errorf("output tokens = %d, want 3000", session.OutputTokens)
	}
	if session.CacheReadTokens != 3000 {
		t.Errorf("cache read = %d, want 3000", session.CacheReadTokens)
	}
	if session.ToolUses["Read"] != 2 {
		t.Errorf("Read tool uses = %d, want 2", session.ToolUses["Read"])
	}
	if session.ToolUses["Bash"] != 1 {
		t.Errorf("Bash tool uses = %d, want 1", session.ToolUses["Bash"])
	}
	if session.ToolErrors != 1 {
		t.Errorf("tool errors = %d, want 1", session.ToolErrors)
	}
	if session.Models["claude-opus-4-6"] != 2 {
		t.Errorf("opus count = %d, want 2", session.Models["claude-opus-4-6"])
	}
	if session.AvgLatencyMs != 2000 {
		t.Errorf("avg latency = %d, want 2000", session.AvgLatencyMs)
	}
}

func TestLoadStatsFromReal(t *testing.T) {
	stats, err := LoadStats()
	if err != nil {
		t.Fatalf("LoadStats() error: %v", err)
	}
	if stats.TotalSessions == 0 {
		t.Error("expected at least 1 session")
	}
	if stats.TotalMessages == 0 {
		t.Error("expected messages > 0")
	}
	if len(stats.Projects) == 0 {
		t.Error("expected at least 1 project")
	}
	if stats.ActiveModel == "" {
		t.Error("expected an active model")
	}
}
