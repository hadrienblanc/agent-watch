package ui

import (
	"strings"
	"testing"
	"time"

	"claude_monitor/internal/data"

	tea "charm.land/bubbletea/v2"
)

func fakeStats() *data.Stats {
	return &data.Stats{
		TotalSessions:     5,
		TotalMessages:     120,
		TotalInputTokens:  50000,
		TotalOutputTokens: 30000,
		TotalCacheRead:    20000,
		TotalToolUses:     45,
		TotalToolErrors:   2,
		ActiveSessions:    1,
		TodaySessions:     3,
		TodayMessages:     60,
		TodayTokens:       40000,
		WeekSessions:      5,
		WeekMessages:      120,
		WeekTokens:        80000,
		ActiveModel:       "claude-opus-4-6",
		Models: map[string]int{
			"claude-opus-4-6":  80,
			"claude-haiku-4-5": 40,
		},
		ToolUsage: map[string]int{
			"Read": 15,
			"Bash": 12,
			"Grep": 10,
			"Edit": 8,
		},
		Sessions: []data.Session{
			{
				ID: "s1", Source: "claude", Slug: "test-session", Project: "test-proj",
				UserMessages: 10, AssistantMessages: 15,
				InputTokens: 25000, OutputTokens: 15000,
				ToolUses: map[string]int{"Read": 5},
				Models:   map[string]int{"claude-opus-4-6": 15},
				StartTime: time.Now().Add(-30 * time.Minute),
				EndTime:   time.Now(),
			},
			{
				ID: "s2", Source: "opencode", Slug: "oc-session", Project: "oc-proj",
				UserMessages: 5, AssistantMessages: 8,
				InputTokens: 8000, OutputTokens: 3000,
				Models: map[string]int{"glm-5": 8},
				ToolUses: map[string]int{},
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now(),
			},
			{
				ID: "s3", Source: "gemini", Slug: "gemini-session", Project: "gemini-proj",
				UserMessages: 3, AssistantMessages: 4,
				InputTokens: 12000, OutputTokens: 2000,
				Models:   map[string]int{"gemini-3-flash-preview": 4},
				ToolUses: map[string]int{"read_file": 2},
				StartTime: time.Now().Add(-45 * time.Minute),
				EndTime:   time.Now(),
			},
		},
		Projects: []data.ProjectSummary{
			{Name: "test-proj", Sessions: 3, Messages: 80, Tokens: 50000, Cost: 1.83},
			{Name: "other-proj", Sessions: 2, Messages: 40, Tokens: 30000, Cost: 1.09},
		},
		DailyCosts: []data.DayCost{
			{Date: time.Now().AddDate(0, 0, -2), InputTokens: 10000, OutputTokens: 5000, CacheRead: 3000, Sessions: 2, Messages: 20, Cost: 0.52},
			{Date: time.Now().AddDate(0, 0, -1), InputTokens: 20000, OutputTokens: 8000, CacheRead: 5000, Sessions: 3, Messages: 40, Cost: 0.95},
			{Date: time.Now(), InputTokens: 15000, OutputTokens: 6000, CacheRead: 4000, Sessions: 1, Messages: 15, Cost: 0.68},
		},
		TotalCost:   2.15,
		LastUpdated: time.Now(),
	}
}

func TestDashboardTabNavigation(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40

	model, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	d = model.(Dashboard)
	if d.tab != 1 {
		t.Errorf("expected tab 1, got %d", d.tab)
	}

	model, _ = d.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	d = model.(Dashboard)
	if d.tab != 2 {
		t.Errorf("expected tab 2, got %d", d.tab)
	}

	// Right arrow
	model, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	d = model.(Dashboard)
	if d.tab != 3 {
		t.Errorf("expected tab 3 after right, got %d", d.tab)
	}

	// Left arrow
	model, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	d = model.(Dashboard)
	if d.tab != 2 {
		t.Errorf("expected tab 2 after left, got %d", d.tab)
	}

	// Left wraps to last tab
	d.tab = 0
	model, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	d = model.(Dashboard)
	if d.tab != 6 {
		t.Errorf("expected tab 6 (wrap left), got %d", d.tab)
	}
}

func TestDashboardViewTabs(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40

	// Overview
	d.tab = 0
	view := d.View()
	if !strings.Contains(view.Content, "Activit") {
		t.Error("overview should contain 'Activité'")
	}
	if !strings.Contains(view.Content, "Tokens") {
		t.Error("overview should contain 'Tokens'")
	}
	if !strings.Contains(view.Content, "semaine") {
		t.Error("overview should contain 'semaine'")
	}
	if !strings.Contains(view.Content, "Sant") {
		t.Error("overview should contain 'Santé'")
	}

	// Sessions
	d.tab = 1
	view = d.View()
	if !strings.Contains(view.Content, "Sessions") {
		t.Error("sessions tab should contain 'Sessions'")
	}

	// Tools
	d.tab = 2
	view = d.View()
	if !strings.Contains(view.Content, "outils") || !strings.Contains(view.Content, "Outil") {
		t.Error("tools tab should contain tool info")
	}

	// Projects
	d.tab = 3
	view = d.View()
	if !strings.Contains(view.Content, "Projet") {
		t.Error("projects tab should contain 'Projet'")
	}
	if !strings.Contains(view.Content, "$1.83") {
		t.Error("projects tab should contain cost per project")
	}

	// Costs - graph
	d.tab = 4
	d.costView = "g"
	view = d.View()
	if !strings.Contains(view.Content, "jour") {
		t.Error("costs tab graph should contain 'jour'")
	}

	// Costs - table
	d.costView = "t"
	view = d.View()
	if !strings.Contains(view.Content, "journalier") {
		t.Error("costs tab table should contain 'journalier'")
	}
	if !strings.Contains(view.Content, "Mar") {
		t.Error("costs tab table should contain date")
	}

	// Sources
	d.tab = 5
	view = d.View()
	if !strings.Contains(view.Content, "claude") {
		t.Error("sources tab should contain 'claude'")
	}
	if !strings.Contains(view.Content, "opencode") {
		t.Error("sources tab should contain 'opencode'")
	}
	if !strings.Contains(view.Content, "gemini") {
		t.Error("sources tab should contain 'gemini'")
	}
	if !strings.Contains(view.Content, "Comparatif") {
		t.Error("sources tab should contain 'Comparatif'")
	}

	// Modèles
	d.tab = 6
	view = d.View()
	if !strings.Contains(view.Content, "claude-opus") {
		t.Error("models tab should contain 'claude-opus'")
	}
	if !strings.Contains(view.Content, "glm-5") {
		t.Error("models tab should contain 'glm-5'")
	}
	if !strings.Contains(view.Content, "gemini-3-flash") {
		t.Error("models tab should contain 'gemini-3-flash'")
	}
}

func TestFmtNum(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{500, "500"},
		{1500, "1.5K"},
		{2340000, "2.3M"},
	}
	for _, tt := range tests {
		got := fmtNum(tt.input)
		if got != tt.want {
			t.Errorf("fmtNum(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFmtDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{30 * time.Second, "30s"},
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1.5h"},
	}
	for _, tt := range tests {
		got := fmtDuration(tt.input)
		if got != tt.want {
			t.Errorf("fmtDuration(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
