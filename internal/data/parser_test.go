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
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard path with tests keyword",
			input:    "-home-hadrienblanc-Projets-tests-form-on-terminal",
			expected: "form-on-terminal",
		},
		{
			name:     "path with two segments before Projets",
			input:    "-home-hadrienblanc-Projets-hadrienblanc-phira",
			expected: "hadrienblanc-phira",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single segment",
			input:    "myproject",
			expected: "myproject",
		},
		{
			name:     "two segments only",
			input:    "a-b",
			expected: "a-b",
		},
		{
			name:     "stops at home keyword but includes segment before",
			input:    "-home-user-myproject",
			expected: "user-myproject",
		},
		{
			name:     "stops at tests keyword",
			input:    "-home-user-tests-myproject",
			expected: "myproject",
		},
		{
			name:     "stops at Projets keyword (case insensitive)",
			input:    "-home-user-Projets-myproject",
			expected: "myproject",
		},
		{
			name:     "stops at projets keyword lowercase",
			input:    "-home-user-projets-myproject",
			expected: "myproject",
		},
		{
			name:     "complex path with nested structure",
			input:    "-home-hadrienblanc-Projets-tests-claude-monitor-internal",
			expected: "claude-monitor-internal",
		},
		{
			name:     "path with only keywords returns last segment",
			input:    "-home-tests-projets",
			expected: "projets",
		},
		{
			name:     "double dash segments - empty segment skipped",
			input:    "-home-user--myproject",
			expected: "user-myproject",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeProjectName(tt.input)
			if result != tt.expected {
				t.Errorf("decodeProjectName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
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
			Type:       "assistant",
			Timestamp:  time.Date(2026, 3, 17, 10, 0, 5, 0, time.UTC),
			SessionID:  "sess-123",
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
			Type:       "assistant",
			Timestamp:  time.Date(2026, 3, 17, 10, 1, 0, 0, time.UTC),
			SessionID:  "sess-123",
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

func TestParseSessionEdgeCases(t *testing.T) {
	t.Run("empty file", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "empty.jsonl")
		os.WriteFile(f, []byte{}, 0644)

		session, err := parseSession(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session == nil {
			t.Fatal("session is nil")
		}
		if session.UserMessages != 0 || session.AssistantMessages != 0 {
			t.Errorf("expected zero messages, got user=%d, assistant=%d",
				session.UserMessages, session.AssistantMessages)
		}
	})

	t.Run("malformed JSON lines", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "malformed.jsonl")
		content := `{"type": "user", "timestamp": "2026-03-17T10:00:00Z"
not valid json at all
{"type": "assistant", "message": {"role": "assistant", "model": "claude-opus-4-6"}}
{"type": "user", "timestamp": "2026-03-17T10:01:00Z", "message": {"role": "user"}}`
		os.WriteFile(f, []byte(content), 0644)

		session, err := parseSession(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should have parsed valid entries despite malformed lines
		if session.UserMessages != 1 {
			t.Errorf("user messages = %d, want 1", session.UserMessages)
		}
	})

	t.Run("entries without messages", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "no-messages.jsonl")
		entries := []Entry{
			{Type: "summary", Timestamp: time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC)},
			{Type: "system", Timestamp: time.Date(2026, 3, 17, 10, 0, 1, 0, time.UTC), SessionID: "no-msg-sess"},
			{Type: "other", Message: nil},
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
			t.Fatalf("unexpected error: %v", err)
		}
		if session.UserMessages != 0 || session.AssistantMessages != 0 {
			t.Errorf("expected zero messages, got user=%d, assistant=%d",
				session.UserMessages, session.AssistantMessages)
		}
		if session.ID != "no-msg-sess" {
			t.Errorf("session ID = %q, want no-msg-sess", session.ID)
		}
	})

	t.Run("only whitespace", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "whitespace.jsonl")
		os.WriteFile(f, []byte("   \n\n  \n"), 0644)

		session, err := parseSession(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session.UserMessages != 0 || session.AssistantMessages != 0 {
			t.Errorf("expected zero messages, got user=%d, assistant=%d",
				session.UserMessages, session.AssistantMessages)
		}
	})
}

func TestComputeCost(t *testing.T) {
	tests := []struct {
		name   string
		model  string
		input  int
		output int
		cacheR int
		cacheW int
		want   float64
	}{
		// Claude models
		{
			name:  "opus basic",
			model: "claude-opus-4-6", input: 1000, output: 500,
			want: 1000*15.0/1_000_000 + 500*75.0/1_000_000,
		},
		{
			name:  "opus with cache",
			model: "claude-opus-4-6", input: 100, output: 500, cacheR: 10000, cacheW: 5000,
			want: 100*15.0/1_000_000 + 500*75.0/1_000_000 + 10000*1.50/1_000_000 + 5000*18.75/1_000_000,
		},
		{
			name:  "sonnet 4-6",
			model: "claude-sonnet-4-6", input: 1000, output: 500, cacheR: 2000, cacheW: 1000,
			want: 1000*3.0/1_000_000 + 500*15.0/1_000_000 + 2000*0.30/1_000_000 + 1000*3.75/1_000_000,
		},
		{
			name:  "haiku 4-5",
			model: "claude-haiku-4-5", input: 5000, output: 2000,
			want: 5000*0.80/1_000_000 + 2000*4.0/1_000_000,
		},
		// GPT models
		{
			name:  "gpt-4o",
			model: "gpt-4o", input: 1000, output: 500, cacheR: 500, cacheW: 200,
			want: 1000*2.50/1_000_000 + 500*10.0/1_000_000 + 500*1.25/1_000_000 + 200*2.50/1_000_000,
		},
		{
			name:  "gpt-4o-mini",
			model: "gpt-4o-mini", input: 10000, output: 5000,
			want: 10000*0.15/1_000_000 + 5000*0.60/1_000_000,
		},
		{
			name:  "gpt-5.4",
			model: "gpt-5.4", input: 2000, output: 1000,
			want: 2000*2.00/1_000_000 + 1000*8.00/1_000_000,
		},
		{
			name:  "o3",
			model: "o3", input: 1500, output: 800, cacheR: 300,
			want: 1500*2.00/1_000_000 + 800*8.00/1_000_000 + 300*0.50/1_000_000,
		},
		// Gemini models
		{
			name:  "gemini-2.5-pro",
			model: "gemini-2.5-pro", input: 1000, output: 500, cacheR: 200, cacheW: 100,
			want: 1000*1.25/1_000_000 + 500*10.0/1_000_000 + 200*0.315/1_000_000 + 100*4.50/1_000_000,
		},
		{
			name:  "gemini-2.5-flash",
			model: "gemini-2.5-flash", input: 5000, output: 3000,
			want: 5000*0.15/1_000_000 + 3000*0.60/1_000_000,
		},
		{
			name:  "gemini-3-flash-preview prefix match",
			model: "gemini-3-flash-preview-20260101", input: 1000, output: 500,
			want: 1000*0.15/1_000_000 + 500*0.60/1_000_000,
		},
		// GLM models
		{
			name:  "glm-5",
			model: "glm-5", input: 1000, output: 500, cacheR: 200, cacheW: 100,
			want: 1000*0.72/1_000_000 + 500*2.30/1_000_000 + 200*0.19/1_000_000 + 100*0.72/1_000_000,
		},
		{
			name:  "glm-4.7-flash",
			model: "glm-4.7-flash", input: 10000, output: 5000,
			want: 10000*0.10/1_000_000 + 5000*0.40/1_000_000,
		},
		{
			name:  "glm-4.5",
			model: "glm-4.5", input: 3000, output: 1500, cacheR: 500,
			want: 3000*0.60/1_000_000 + 1500*2.20/1_000_000 + 500*0.11/1_000_000,
		},
		// MiniMax models
		{
			name:  "MiniMax-M2.5",
			model: "MiniMax-M2.5", input: 5000, output: 2500, cacheR: 1000, cacheW: 500,
			want: 5000*0.30/1_000_000 + 2500*1.20/1_000_000 + 1000*0.03/1_000_000 + 500*0.375/1_000_000,
		},
		// Edge cases
		{
			name:  "zero tokens",
			model: "claude-opus-4-6",
			want:  0,
		},
		{
			name:  "unknown model falls back to opus pricing",
			model: "unknown-model-xyz", input: 1000, output: 500,
			want: 1000*15.0/1_000_000 + 500*75.0/1_000_000,
		},
		{
			name:  "synthetic model costs nothing",
			model: "<synthetic>", input: 10000, output: 5000, cacheR: 3000,
			want: 0,
		},
		{
			name:  "only cache costs",
			model: "claude-sonnet-4-5", cacheR: 10000, cacheW: 5000,
			want: 10000*0.30/1_000_000 + 5000*3.75/1_000_000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeCost(tt.model, tt.input, tt.output, tt.cacheR, tt.cacheW)
			if diff := got - tt.want; diff > 0.000001 || diff < -0.000001 {
				t.Errorf("ComputeCost(%s, %d, %d, %d, %d) = %f, want %f",
					tt.model, tt.input, tt.output, tt.cacheR, tt.cacheW, got, tt.want)
			}
		})
	}
}

func TestPricingFor(t *testing.T) {
	// Exact match
	p := pricingFor("claude-opus-4-6")
	if p.InputPerM != 15.0 {
		t.Errorf("opus input pricing = %f, want 15.0", p.InputPerM)
	}

	// Prefix match
	p = pricingFor("claude-opus-4-6-20260301")
	if p.InputPerM != 15.0 {
		t.Errorf("opus prefix match input = %f, want 15.0", p.InputPerM)
	}

	// Unknown model -> default (opus)
	p = pricingFor("totally-unknown")
	if p.InputPerM != 15.0 {
		t.Errorf("unknown model should default to opus, got input = %f", p.InputPerM)
	}

	// Synthetic
	p = pricingFor("<synthetic>")
	if p.InputPerM != 0 {
		t.Errorf("synthetic input = %f, want 0", p.InputPerM)
	}
}

func TestDayKeyRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
	}{
		{
			name: "UTC midnight",
			time: time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "UTC with time",
			time: time.Date(2026, 3, 17, 14, 30, 45, 123, time.UTC),
		},
		{
			name: "local timezone",
			time: time.Date(2026, 12, 31, 23, 59, 59, 999999999, time.Local),
		},
		{
			name: "beginning of year",
			time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "leap year date",
			time: time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "end of month",
			time: time.Date(2026, 1, 31, 15, 45, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert time to dayKey and back
			dk := dayKeyFrom(tt.time)
			result := dk.Time()

			// The result should be midnight of the same day in Local timezone
			expectedYear, expectedMonth, expectedDay := tt.time.Date()
			resultYear, resultMonth, resultDay := result.Date()

			if resultYear != expectedYear {
				t.Errorf("year = %d, want %d", resultYear, expectedYear)
			}
			if resultMonth != expectedMonth {
				t.Errorf("month = %d, want %d", resultMonth, expectedMonth)
			}
			if resultDay != expectedDay {
				t.Errorf("day = %d, want %d", resultDay, expectedDay)
			}

			// Verify time is truncated to midnight
			h, m, s := result.Clock()
			if h != 0 || m != 0 || s != 0 {
				t.Errorf("time should be midnight, got %02d:%02d:%02d", h, m, s)
			}
		})
	}
}

func TestDayKeyFromAndTimeConsistency(t *testing.T) {
	// Test that dayKeyFrom -> Time -> dayKeyFrom is idempotent
	original := time.Date(2026, 3, 17, 10, 30, 45, 123456789, time.Local)

	dk1 := dayKeyFrom(original)
	midnight := dk1.Time()
	dk2 := dayKeyFrom(midnight)

	if dk1 != dk2 {
		t.Errorf("dayKey not idempotent: dk1=%v, dk2=%v", dk1, dk2)
	}

	// Apply Time() again should give same result
	midnight2 := dk2.Time()
	if !midnight.Equal(midnight2) {
		t.Errorf("Time() not idempotent: midnight=%v, midnight2=%v", midnight, midnight2)
	}
}

func TestAggregateSession(t *testing.T) {
	tests := []struct {
		name           string
		session        *Session
		wantInput      int
		wantOutput     int
		wantCache      int
		wantMessages   int
		wantToolErrors int
		wantTools      map[string]int
		wantModels     map[string]int
	}{
		{
			name: "single session with all fields",
			session: &Session{
				InputTokens:         5000,
				OutputTokens:        2000,
				CacheReadTokens:     1000,
				UserMessages:        3,
				AssistantMessages:   5,
				ToolErrors:          2,
				ToolUses:            map[string]int{"Read": 4, "Bash": 2},
				Models:              map[string]int{"claude-opus-4-6": 5},
			},
			wantInput:      5000,
			wantOutput:     2000,
			wantCache:      1000,
			wantMessages:   8,
			wantToolErrors: 2,
			wantTools:      map[string]int{"Read": 4, "Bash": 2},
			wantModels:     map[string]int{"claude-opus-4-6": 5},
		},
		{
			name: "session with zero values",
			session: &Session{
				ToolUses: make(map[string]int),
				Models:   make(map[string]int),
			},
			wantInput:      0,
			wantOutput:     0,
			wantCache:      0,
			wantMessages:   0,
			wantToolErrors: 0,
			wantTools:      map[string]int{},
			wantModels:     map[string]int{},
		},
		{
			name: "session with large numbers",
			session: &Session{
				InputTokens:         1_000_000,
				OutputTokens:        500_000,
				CacheReadTokens:     250_000,
				UserMessages:        100,
				AssistantMessages:   150,
				ToolErrors:          10,
				ToolUses:            map[string]int{"Tool1": 50, "Tool2": 100, "Tool3": 75},
				Models:              map[string]int{"model-a": 80, "model-b": 70},
			},
			wantInput:      1_000_000,
			wantOutput:     500_000,
			wantCache:      250_000,
			wantMessages:   250,
			wantToolErrors: 10,
			wantTools:      map[string]int{"Tool1": 50, "Tool2": 100, "Tool3": 75},
			wantModels:     map[string]int{"model-a": 80, "model-b": 70},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &Stats{
				ToolUsage: make(map[string]int),
				Models:    make(map[string]int),
			}

			aggregateSession(stats, tt.session)

			if stats.TotalInputTokens != tt.wantInput {
				t.Errorf("TotalInputTokens = %d, want %d", stats.TotalInputTokens, tt.wantInput)
			}
			if stats.TotalOutputTokens != tt.wantOutput {
				t.Errorf("TotalOutputTokens = %d, want %d", stats.TotalOutputTokens, tt.wantOutput)
			}
			if stats.TotalCacheRead != tt.wantCache {
				t.Errorf("TotalCacheRead = %d, want %d", stats.TotalCacheRead, tt.wantCache)
			}
			if stats.TotalMessages != tt.wantMessages {
				t.Errorf("TotalMessages = %d, want %d", stats.TotalMessages, tt.wantMessages)
			}
			if stats.TotalToolErrors != tt.wantToolErrors {
				t.Errorf("TotalToolErrors = %d, want %d", stats.TotalToolErrors, tt.wantToolErrors)
			}

			for tool, wantCount := range tt.wantTools {
				if stats.ToolUsage[tool] != wantCount {
					t.Errorf("ToolUsage[%s] = %d, want %d", tool, stats.ToolUsage[tool], wantCount)
				}
			}
			for model, wantCount := range tt.wantModels {
				if stats.Models[model] != wantCount {
					t.Errorf("Models[%s] = %d, want %d", model, stats.Models[model], wantCount)
				}
			}
		})
	}
}

func TestAggregateSessionAccumulates(t *testing.T) {
	// Test that calling aggregateSession multiple times accumulates values
	stats := &Stats{
		ToolUsage: make(map[string]int),
		Models:    make(map[string]int),
	}

	sessions := []*Session{
		{
			InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 200,
			UserMessages: 1, AssistantMessages: 2, ToolErrors: 1,
			ToolUses: map[string]int{"Read": 1}, Models: map[string]int{"model-a": 1},
		},
		{
			InputTokens: 2000, OutputTokens: 1000, CacheReadTokens: 400,
			UserMessages: 2, AssistantMessages: 3, ToolErrors: 2,
			ToolUses: map[string]int{"Read": 2, "Bash": 1}, Models: map[string]int{"model-a": 2, "model-b": 1},
		},
		{
			InputTokens: 500, OutputTokens: 250, CacheReadTokens: 100,
			UserMessages: 1, AssistantMessages: 1, ToolErrors: 0,
			ToolUses: map[string]int{"Bash": 3}, Models: map[string]int{"model-b": 1},
		},
	}

	for _, s := range sessions {
		aggregateSession(stats, s)
	}

	if stats.TotalInputTokens != 3500 {
		t.Errorf("TotalInputTokens = %d, want 3500", stats.TotalInputTokens)
	}
	if stats.TotalOutputTokens != 1750 {
		t.Errorf("TotalOutputTokens = %d, want 1750", stats.TotalOutputTokens)
	}
	if stats.TotalCacheRead != 700 {
		t.Errorf("TotalCacheRead = %d, want 700", stats.TotalCacheRead)
	}
	if stats.TotalMessages != 10 {
		t.Errorf("TotalMessages = %d, want 10", stats.TotalMessages)
	}
	if stats.TotalToolErrors != 3 {
		t.Errorf("TotalToolErrors = %d, want 3", stats.TotalToolErrors)
	}
	if stats.ToolUsage["Read"] != 3 {
		t.Errorf("ToolUsage[Read] = %d, want 3", stats.ToolUsage["Read"])
	}
	if stats.ToolUsage["Bash"] != 4 {
		t.Errorf("ToolUsage[Bash] = %d, want 4", stats.ToolUsage["Bash"])
	}
	if stats.Models["model-a"] != 3 {
		t.Errorf("Models[model-a] = %d, want 3", stats.Models["model-a"])
	}
	if stats.Models["model-b"] != 2 {
		t.Errorf("Models[model-b] = %d, want 2", stats.Models["model-b"])
	}
	if stats.TotalToolUses != 7 {
		t.Errorf("TotalToolUses = %d, want 7", stats.TotalToolUses)
	}
}

func TestAggregateProject(t *testing.T) {
	tests := []struct {
		name            string
		sessions        []*Session
		wantSessions    int
		wantMessages    int
		wantTokens      int
		wantInput       int
		wantOutput      int
		wantCacheRead   int
		wantCost        float64
	}{
		{
			name: "single session",
			sessions: []*Session{
				{
					UserMessages:      5,
					AssistantMessages: 10,
					InputTokens:       10000,
					OutputTokens:      5000,
					CacheReadTokens:   2000,
					Cost:              0.25,
				},
			},
			wantSessions:  1,
			wantMessages:  15,
			wantTokens:    15000,
			wantInput:     10000,
			wantOutput:    5000,
			wantCacheRead: 2000,
			wantCost:      0.25,
		},
		{
			name: "multiple sessions accumulate",
			sessions: []*Session{
				{UserMessages: 2, AssistantMessages: 3, InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 100, Cost: 0.01},
				{UserMessages: 5, AssistantMessages: 7, InputTokens: 2000, OutputTokens: 1000, CacheReadTokens: 200, Cost: 0.02},
				{UserMessages: 1, AssistantMessages: 1, InputTokens: 500, OutputTokens: 250, CacheReadTokens: 50, Cost: 0.005},
			},
			wantSessions:  3,
			wantMessages:  19,
			wantTokens:    5250,
			wantInput:     3500,
			wantOutput:    1750,
			wantCacheRead: 350,
			wantCost:      0.035,
		},
		{
			name:            "empty sessions slice",
			sessions:        []*Session{},
			wantSessions:    0,
			wantMessages:    0,
			wantTokens:      0,
			wantInput:       0,
			wantOutput:      0,
			wantCacheRead:   0,
			wantCost:        0,
		},
		{
			name: "session with zero values",
			sessions: []*Session{
				{UserMessages: 0, AssistantMessages: 0, InputTokens: 0, OutputTokens: 0, CacheReadTokens: 0, Cost: 0},
			},
			wantSessions:  1,
			wantMessages:  0,
			wantTokens:    0,
			wantInput:     0,
			wantOutput:    0,
			wantCacheRead: 0,
			wantCost:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &ProjectSummary{}
			for _, s := range tt.sessions {
				aggregateProject(ps, s)
			}

			if ps.Sessions != tt.wantSessions {
				t.Errorf("Sessions = %d, want %d", ps.Sessions, tt.wantSessions)
			}
			if ps.Messages != tt.wantMessages {
				t.Errorf("Messages = %d, want %d", ps.Messages, tt.wantMessages)
			}
			if ps.Tokens != tt.wantTokens {
				t.Errorf("Tokens = %d, want %d", ps.Tokens, tt.wantTokens)
			}
			if ps.InputTokens != tt.wantInput {
				t.Errorf("InputTokens = %d, want %d", ps.InputTokens, tt.wantInput)
			}
			if ps.OutputTokens != tt.wantOutput {
				t.Errorf("OutputTokens = %d, want %d", ps.OutputTokens, tt.wantOutput)
			}
			if ps.CacheRead != tt.wantCacheRead {
				t.Errorf("CacheRead = %d, want %d", ps.CacheRead, tt.wantCacheRead)
			}
			if diff := ps.Cost - tt.wantCost; diff > 0.000001 || diff < -0.000001 {
				t.Errorf("Cost = %f, want %f", ps.Cost, tt.wantCost)
			}
		})
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
				Model:  "gemini-3-flash-preview",
				Tokens: &geminiTokens{Input: 6505, Output: 81, Cached: 100, Thoughts: 101, Total: 6787},
				ToolCalls: []struct {
					Name string `json:"name"`
				}{
					{Name: "read_file"},
					{Name: "list_files"},
				},
			},
			{Type: "user", Timestamp: "2026-03-17T15:49:35.000Z"},
			{
				Type: "gemini", Timestamp: "2026-03-17T15:50:00.000Z",
				Model:  "gemini-3-flash-preview",
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
	// We know there are gemini sessions on this machine
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

	// Verify that gemini sources are included
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

func TestStatsJSONSerializable(t *testing.T) {
	stats := &Stats{
		Sessions: []Session{
			{
				ID: "s1", Source: "claude", Project: "proj",
				UserMessages: 5, AssistantMessages: 3,
				InputTokens: 1000, OutputTokens: 500,
				ToolUses: map[string]int{"Read": 2},
				Models:   map[string]int{"opus": 3},
				PerDay: map[dayKey]*DayTokens{
					dayKeyFrom(time.Now()): {Input: 1000, Output: 500},
				},
			},
		},
		ToolUsage:    map[string]int{"Read": 2},
		Models:       map[string]int{"opus": 3},
		TotalCost:    1.50,
		LastUpdated:  time.Now(),
		DailyCosts:   []DayCost{{Date: time.Now(), Cost: 1.50}},
	}

	// PerDay has map[dayKey] which is not JSON-serializable.
	// json:"-" tag must exclude it so encoding succeeds.
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("json.Marshal(Stats) failed: %v", err)
	}

	// Roundtrip: unmarshal into a new Stats
	var decoded Stats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(Stats) failed: %v", err)
	}

	if decoded.TotalCost != 1.50 {
		t.Errorf("TotalCost = %.2f, want 1.50", decoded.TotalCost)
	}
	if len(decoded.Sessions) != 1 {
		t.Errorf("Sessions count = %d, want 1", len(decoded.Sessions))
	}
	// PerDay should be nil after roundtrip (excluded from JSON)
	if decoded.Sessions[0].PerDay != nil {
		t.Error("PerDay should be nil after JSON roundtrip (json:\"-\")")
	}
}
