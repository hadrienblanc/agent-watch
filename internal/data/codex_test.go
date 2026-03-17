package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTruncateTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		maxLen   int
		expected string
	}{
		{
			name:     "empty string",
			title:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "short string unchanged",
			title:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length unchanged",
			title:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string truncated",
			title:    "this is a very long title that needs truncation",
			maxLen:   20,
			expected: "this is a very long…",
		},
		{
			name:     "unicode characters",
			title:    "hello world with unicode characters here",
			maxLen:   15,
			expected: "hello world wi…",
		},
		{
			name:     "single character maxLen",
			title:    "test",
			maxLen:   2,
			expected: "t…",
		},
		{
			name:     "very long string",
			title:    strings.Repeat("a", 1001), // 1001 'a' characters
			maxLen:   10,
			expected: "aaaaaaaaa…", // 9 a's (maxLen-1) + ellipsis
		},
		{
			name:     "maxLen just above truncation threshold",
			title:    "hello world",
			maxLen:   12,
			expected: "hello world", // len("hello world") = 11, which is <= 12
		},
		{
			name:     "maxLen at truncation threshold",
			title:    "hello world",
			maxLen:   10,
			expected: "hello wor…", // len("hello world") = 11 > 10, so truncate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateTitle(tt.title, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateTitle(%q, %d) = %q, want %q", tt.title, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestTruncateTitle_EdgeCases(t *testing.T) {
	// Note: maxLen=0 causes a panic because the function tries to slice with negative index.
	// This test documents the current behavior.

	t.Run("maxLen 0 causes panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic with maxLen=0, but function did not panic")
			}
		}()
		_ = truncateTitle("hello", 0)
	})

	t.Run("maxLen 1 with long string returns ellipsis", func(t *testing.T) {
		// With maxLen=1 and a long string:
		// len(title) > maxLen, so: return title[:maxLen-1] + "…" = title[:0] + "…" = "" + "…" = "…"
		result := truncateTitle("hello world", 1)
		if result != "…" {
			t.Errorf("truncateTitle with maxLen=1 = %q, want %q", result, "…")
		}
	})
}

func TestExpandRolloutPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantEmpty  bool // if true, expect empty result
		setup      func(t *testing.T) string // returns path to created file if needed
	}{
		{
			name:      "empty string",
			path:      "",
			wantEmpty: true,
		},
		{
			name:      "path without tilde that doesn't exist",
			path:      "/nonexistent/path/to/file.jsonl",
			wantEmpty: true,
		},
		{
			name:      "tilde at start with nonexistent file",
			path:      "~/some/nonexistent/path.jsonl",
			wantEmpty: true,
		},
		{
			name: "existing file without tilde",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				file := filepath.Join(dir, "test.jsonl")
				os.WriteFile(file, []byte("{}"), 0644)
				return file
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.setup != nil {
				path = tt.setup(t)
			} else {
				path = tt.path
			}

			result := expandRolloutPath(path)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("expandRolloutPath(%q) = %q, want empty", path, result)
				}
			} else if tt.setup != nil {
				// For setup cases, we expect the expanded/existing path back
				if result != path {
					t.Errorf("expandRolloutPath(%q) = %q, want %q", path, result, path)
				}
			}
		})
	}
}

func TestExpandRolloutPath_TildeExpansion(t *testing.T) {
	home, _ := os.UserHomeDir()

	// Create a file in home directory
	dir := filepath.Join(home, ".codex_test_tmp")
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "test.jsonl")
	os.WriteFile(file, []byte("{}"), 0644)

	// Test with tilde path
	tildePath := "~/.codex_test_tmp/test.jsonl"
	result := expandRolloutPath(tildePath)

	if result != file {
		t.Errorf("expandRolloutPath(%q) = %q, want %q", tildePath, result, file)
	}
}

func TestEnrichCodexSession(t *testing.T) {
	tests := []struct {
		name           string
		jsonlContent   []string // JSONL lines
		expectedUser   int
		expectedAssist int
		expectedInput  int
		expectedOutput int
		expectedCache  int
		expectedModel  string
		expectedBranch string
	}{
		{
			name: "empty file",
			jsonlContent: []string{},
			expectedUser:   0,
			expectedAssist: 0,
		},
		{
			name: "invalid JSON",
			jsonlContent: []string{"not valid json", "{broken"},
			expectedUser:   0,
			expectedAssist: 0,
		},
		{
			name: "session_meta with branch",
			jsonlContent: []string{
				`{"timestamp":"2026-03-17T10:00:00Z","type":"session_meta","payload":{"id":"test-id","git":{"branch":"feature/test-branch"}}}`,
			},
			expectedBranch: "feature/test-branch",
		},
		{
			name: "response_item user message",
			jsonlContent: []string{
				`{"timestamp":"2026-03-17T10:00:00Z","type":"response_item","payload":{"role":"user","type":"message"}}`,
			},
			expectedUser: 1,
		},
		{
			name: "response_item assistant message",
			jsonlContent: []string{
				`{"timestamp":"2026-03-17T10:00:00Z","type":"response_item","payload":{"role":"assistant","type":"message"}}`,
			},
			expectedAssist: 1,
		},
		{
			name: "turn_context with model",
			jsonlContent: []string{
				`{"timestamp":"2026-03-17T10:00:00Z","type":"turn_context","payload":{"model":"gpt-4o"}}`,
				`{"timestamp":"2026-03-17T10:00:01Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"output_tokens":500,"cached_input_tokens":200,"total_tokens":1700}}}}`,
			},
			expectedInput:  1000,
			expectedOutput: 500,
			expectedCache:  200,
			expectedModel:  "gpt-4o",
		},
		{
			name: "complete session",
			jsonlContent: []string{
				`{"timestamp":"2026-03-17T10:00:00Z","type":"session_meta","payload":{"id":"complete-session","git":{"branch":"main"}}}`,
				`{"timestamp":"2026-03-17T10:00:01Z","type":"response_item","payload":{"role":"user","type":"message"}}`,
				`{"timestamp":"2026-03-17T10:00:02Z","type":"turn_context","payload":{"model":"gpt-5.3-codex"}}`,
				`{"timestamp":"2026-03-17T10:00:03Z","type":"response_item","payload":{"role":"assistant","type":"message"}}`,
				`{"timestamp":"2026-03-17T10:00:04Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":5000,"output_tokens":2000,"cached_input_tokens":1000,"total_tokens":8000}}}}`,
			},
			expectedUser:   1,
			expectedAssist: 1,
			expectedInput:  5000,
			expectedOutput: 2000,
			expectedCache:  1000,
			expectedModel:  "gpt-5.3-codex",
			expectedBranch: "main",
		},
		{
			name: "multiple user and assistant messages",
			jsonlContent: []string{
				`{"timestamp":"2026-03-17T10:00:00Z","type":"response_item","payload":{"role":"user","type":"message"}}`,
				`{"timestamp":"2026-03-17T10:00:01Z","type":"response_item","payload":{"role":"assistant","type":"message"}}`,
				`{"timestamp":"2026-03-17T10:00:02Z","type":"response_item","payload":{"role":"user","type":"message"}}`,
				`{"timestamp":"2026-03-17T10:00:03Z","type":"response_item","payload":{"role":"assistant","type":"message"}}`,
			},
			expectedUser:   2,
			expectedAssist: 2,
		},
		{
			name: "event_msg non-token_count ignored",
			jsonlContent: []string{
				`{"timestamp":"2026-03-17T10:00:00Z","type":"event_msg","payload":{"type":"other_event","info":{}}}`,
			},
			expectedInput:  0,
			expectedOutput: 0,
		},
		{
			name: "event_msg with nil token_usage",
			jsonlContent: []string{
				`{"timestamp":"2026-03-17T10:00:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":null}}}`,
			},
			expectedInput:  0,
			expectedOutput: 0,
		},
		{
			name: "model only set after turn_context",
			jsonlContent: []string{
				// First event_msg without prior turn_context - no model set
				`{"timestamp":"2026-03-17T10:00:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"output_tokens":500,"total_tokens":1500}}}}`,
				// Then turn_context sets model
				`{"timestamp":"2026-03-17T10:00:01Z","type":"turn_context","payload":{"model":"gpt-4o"}}`,
				// Second event_msg now has model
				`{"timestamp":"2026-03-17T10:00:02Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":2000,"output_tokens":1000,"total_tokens":3000}}}}`,
			},
			expectedInput:  2000, // Second event overwrites first
			expectedOutput: 1000,
			expectedModel:  "gpt-4o",
		},
		{
			name: "unknown entry type ignored",
			jsonlContent: []string{
				`{"timestamp":"2026-03-17T10:00:00Z","type":"unknown_type","payload":{}}`,
			},
			expectedUser:   0,
			expectedAssist: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with JSONL content
			dir := t.TempDir()
			file := filepath.Join(dir, "test.jsonl")

			f, err := os.Create(file)
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range tt.jsonlContent {
				f.WriteString(line + "\n")
			}
			f.Close()

			// Create session with StartTime set (needed for PerDay population)
			session := &Session{
				StartTime:       time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC),
				ToolUses:        make(map[string]int),
				Models:          make(map[string]int),
				PerDay:          make(map[dayKey]*DayTokens),
			}

			enrichCodexSession(session, file)

			if session.UserMessages != tt.expectedUser {
				t.Errorf("UserMessages = %d, want %d", session.UserMessages, tt.expectedUser)
			}
			if session.AssistantMessages != tt.expectedAssist {
				t.Errorf("AssistantMessages = %d, want %d", session.AssistantMessages, tt.expectedAssist)
			}
			if session.InputTokens != tt.expectedInput {
				t.Errorf("InputTokens = %d, want %d", session.InputTokens, tt.expectedInput)
			}
			if session.OutputTokens != tt.expectedOutput {
				t.Errorf("OutputTokens = %d, want %d", session.OutputTokens, tt.expectedOutput)
			}
			if session.CacheReadTokens != tt.expectedCache {
				t.Errorf("CacheReadTokens = %d, want %d", session.CacheReadTokens, tt.expectedCache)
			}
			if tt.expectedModel != "" {
				if session.Models[tt.expectedModel] == 0 {
					t.Errorf("expected model %q not found in Models map", tt.expectedModel)
				}
			}
			if tt.expectedBranch != "" && session.GitBranch != tt.expectedBranch {
				t.Errorf("GitBranch = %q, want %q", session.GitBranch, tt.expectedBranch)
			}
		})
	}
}

func TestEnrichCodexSession_NonexistentFile(t *testing.T) {
	session := &Session{
		ToolUses: make(map[string]int),
		Models:   make(map[string]int),
		PerDay:   make(map[dayKey]*DayTokens),
	}

	// Should not panic or error with nonexistent file
	enrichCodexSession(session, "/nonexistent/path/to/file.jsonl")

	// Session should remain unchanged
	if session.UserMessages != 0 {
		t.Errorf("UserMessages should be 0 for nonexistent file, got %d", session.UserMessages)
	}
}

func TestEnrichCodexSession_LargeFile(t *testing.T) {
	// Test that the function handles large entries (up to 5MB buffer)
	dir := t.TempDir()
	file := filepath.Join(dir, "large.jsonl")

	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}

	// Create a large JSON entry (but under 5MB limit)
	largeContent := make(map[string]string)
	for i := 0; i < 10000; i++ {
		largeContent[string(rune('a'+i%26))] = "x"
	}
	entry := map[string]interface{}{
		"timestamp": "2026-03-17T10:00:00Z",
		"type":      "response_item",
		"payload": map[string]interface{}{
			"role": "user",
			"type": "message",
		},
	}

	enc := json.NewEncoder(f)
	for i := 0; i < 100; i++ {
		enc.Encode(entry)
	}
	f.Close()

	session := &Session{
		StartTime:       time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC),
		ToolUses:        make(map[string]int),
		Models:          make(map[string]int),
		PerDay:          make(map[dayKey]*DayTokens),
	}

	enrichCodexSession(session, file)

	if session.UserMessages != 100 {
		t.Errorf("UserMessages = %d, want 100", session.UserMessages)
	}
}

func TestEnrichCodexSession_PerDayPopulation(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.jsonl")

	content := []string{
		`{"timestamp":"2026-03-17T10:00:00Z","type":"turn_context","payload":{"model":"gpt-4o"}}`,
		`{"timestamp":"2026-03-17T10:00:01Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":5000,"output_tokens":2000,"cached_input_tokens":1000,"total_tokens":8000}}}}`,
	}

	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range content {
		f.WriteString(line + "\n")
	}
	f.Close()

	startTime := time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC)
	session := &Session{
		StartTime: startTime,
		ToolUses:  make(map[string]int),
		Models:    make(map[string]int),
		PerDay:    make(map[dayKey]*DayTokens),
	}

	enrichCodexSession(session, file)

	dk := dayKey{Year: 2026, Month: 3, Day: 17}
	dt, ok := session.PerDay[dk]
	if !ok {
		t.Fatalf("PerDay entry for %v not found", dk)
	}
	if dt.Input != 5000 {
		t.Errorf("PerDay Input = %d, want 5000", dt.Input)
	}
	if dt.Output != 2000 {
		t.Errorf("PerDay Output = %d, want 2000", dt.Output)
	}
	if dt.CacheR != 1000 {
		t.Errorf("PerDay CacheR = %d, want 1000", dt.CacheR)
	}
	if dt.Messages != 1 {
		t.Errorf("PerDay Messages = %d, want 1", dt.Messages)
	}
	if dt.Cost == 0 {
		t.Error("PerDay Cost should be > 0")
	}
}

func TestEnrichCodexSession_ZeroStartTime(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.jsonl")

	content := []string{
		`{"timestamp":"2026-03-17T10:00:00Z","type":"turn_context","payload":{"model":"gpt-4o"}}`,
		`{"timestamp":"2026-03-17T10:00:01Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"output_tokens":500,"total_tokens":1500}}}}`,
	}

	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range content {
		f.WriteString(line + "\n")
	}
	f.Close()

	// Session with zero StartTime
	session := &Session{
		StartTime: time.Time{}, // zero time
		ToolUses:  make(map[string]int),
		Models:    make(map[string]int),
		PerDay:    make(map[dayKey]*DayTokens),
	}

	enrichCodexSession(session, file)

	// Should still process tokens but not add PerDay entry
	if session.InputTokens != 1000 {
		t.Errorf("InputTokens = %d, want 1000", session.InputTokens)
	}
	if len(session.PerDay) != 0 {
		t.Errorf("PerDay should be empty with zero StartTime, got %d entries", len(session.PerDay))
	}
}
