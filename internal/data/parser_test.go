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

func TestParseGeminiSession(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "session-2026-03-17T15-49-test.json")

	session := geminiSession{
		SessionID:   "test-gemini-id",
		StartTime:   "2026-03-17T15:49:23.734Z",
		LastUpdated: "2026-03-17T15:50:00.000Z",
		Messages: []geminiMessage{
			{Type: "user", Timestamp: "2026-03-17T15:49:23.734Z"},
			{
				Type: "gemini", Timestamp: "2026-03-17T15:49:30.000Z",
				Model: "gemini-3-flash-preview",
				Tokens: &geminiTokens{Input: 6505, Output: 81, Cached: 100, Thoughts: 101, Total: 6787},
				ToolCalls: []struct{ Name string `json:"name"` }{
					{Name: "read_file"},
					{Name: "list_files"},
				},
			},
			{Type: "user", Timestamp: "2026-03-17T15:49:35.000Z"},
			{
				Type: "gemini", Timestamp: "2026-03-17T15:50:00.000Z",
				Model: "gemini-3-flash-preview",
				Tokens: &geminiTokens{Input: 8000, Output: 200, Total: 8200},
			},
		},
	}

	data, _ := json.Marshal(session)
	os.WriteFile(f, data, 0644)

	s, err := parseGeminiSession(f, "test-proj")
	if err != nil {
		t.Fatal(err)
	}
	if s.Source != "gemini" {
		t.Errorf("source = %q, want gemini", s.Source)
	}
	if s.UserMessages != 2 {
		t.Errorf("user messages = %d, want 2", s.UserMessages)
	}
	if s.AssistantMessages != 2 {
		t.Errorf("assistant messages = %d, want 2", s.AssistantMessages)
	}
	if s.InputTokens != 14505 {
		t.Errorf("input tokens = %d, want 14505", s.InputTokens)
	}
	if s.OutputTokens != 281 {
		t.Errorf("output tokens = %d, want 281", s.OutputTokens)
	}
	if s.CacheReadTokens != 100 {
		t.Errorf("cache read = %d, want 100", s.CacheReadTokens)
	}
	if s.ToolUses["read_file"] != 1 {
		t.Errorf("read_file tool uses = %d, want 1", s.ToolUses["read_file"])
	}
	if s.ToolUses["list_files"] != 1 {
		t.Errorf("list_files tool uses = %d, want 1", s.ToolUses["list_files"])
	}
	if s.Models["gemini-3-flash-preview"] != 2 {
		t.Errorf("model count = %d, want 2", s.Models["gemini-3-flash-preview"])
	}
	if s.Project != "test-proj" {
		t.Errorf("project = %q, want test-proj", s.Project)
	}
}

func TestLoadGeminiSessions(t *testing.T) {
	sessions, err := LoadGeminiSessions()
	if err != nil {
		t.Fatalf("LoadGeminiSessions() error: %v", err)
	}
	// On sait qu'il y a des sessions gemini sur cette machine
	if len(sessions) == 0 {
		t.Skip("no gemini sessions found")
	}
	for _, s := range sessions {
		if s.Source != "gemini" {
			t.Errorf("session %s source = %q, want gemini", s.ID, s.Source)
		}
		if len(s.Models) == 0 {
			t.Errorf("session %s has no models", s.ID)
		}
	}
	t.Logf("Found %d gemini sessions", len(sessions))
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

	// Vérifier que les sources gemini sont incluses
	hasGemini := false
	for _, s := range stats.Sessions {
		if s.Source == "gemini" {
			hasGemini = true
			break
		}
	}
	if !hasGemini {
		t.Error("expected gemini sessions in stats")
	}
}
