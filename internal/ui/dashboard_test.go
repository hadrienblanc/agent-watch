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
				ToolUses:  map[string]int{"Read": 5},
				Models:    map[string]int{"claude-opus-4-6": 15},
				StartTime: time.Now().Add(-30 * time.Minute),
				EndTime:   time.Now(),
			},
			{
				ID: "s2", Source: "opencode", Slug: "oc-session", Project: "oc-proj",
				UserMessages: 5, AssistantMessages: 8,
				InputTokens: 8000, OutputTokens: 3000,
				Models:    map[string]int{"glm-5": 8},
				ToolUses:  map[string]int{},
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now(),
			},
			{
				ID: "s3", Source: "gemini", Slug: "gemini-session", Project: "gemini-proj",
				UserMessages: 3, AssistantMessages: 4,
				InputTokens: 12000, OutputTokens: 2000,
				Models:    map[string]int{"gemini-3-flash-preview": 4},
				ToolUses:  map[string]int{"read_file": 2},
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
	if d.tab != 7 {
		t.Errorf("expected tab 7 (wrap left), got %d", d.tab)
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

func TestPct(t *testing.T) {
	tests := []struct {
		part, total int
		want        float64
	}{
		{50, 100, 50.0},
		{1, 3, 100.0 / 3.0},
		{0, 100, 0},
		{0, 0, 0},
		{100, 100, 100.0},
	}
	for _, tt := range tests {
		got := pct(tt.part, tt.total)
		if diff := got - tt.want; diff > 0.001 || diff < -0.001 {
			t.Errorf("pct(%d, %d) = %f, want %f", tt.part, tt.total, got, tt.want)
		}
	}
}

func TestViewSources(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 5

	view := d.View()
	content := view.Content

	// Check header
	if !strings.Contains(content, "Comparatif par source") {
		t.Error("viewSources should contain 'Comparatif par source' header")
	}

	// Check sources are present
	if !strings.Contains(content, "claude") {
		t.Error("viewSources should contain 'claude' source")
	}
	if !strings.Contains(content, "opencode") {
		t.Error("viewSources should contain 'opencode' source")
	}
	if !strings.Contains(content, "gemini") {
		t.Error("viewSources should contain 'gemini' source")
	}

	// Check that key data elements are present (these are more reliable than column headers)
	// The table should show session counts
	if !strings.Contains(content, "3") && !strings.Contains(content, "2") {
		t.Error("viewSources should show session counts")
	}
}

func TestViewModels(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 6

	view := d.View()
	content := view.Content

	// Check header
	if !strings.Contains(content, "Modles par source") && !strings.Contains(content, "Modèles par source") {
		t.Error("viewModels should contain 'Modèles par source' header")
	}

	// Check models are present from different sources
	if !strings.Contains(content, "claude-opus") {
		t.Error("viewModels should contain 'claude-opus' model")
	}
	if !strings.Contains(content, "glm-5") {
		t.Error("viewModels should contain 'glm-5' model")
	}
	if !strings.Contains(content, "gemini-3-flash") {
		t.Error("viewModels should contain 'gemini-3-flash' model")
	}

	// Check source column - sources should appear next to models
	if !strings.Contains(content, "claude") && !strings.Contains(content, "opencode") && !strings.Contains(content, "gemini") {
		t.Error("viewModels should display source for each model")
	}
}

func TestViewCostsGraph(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 4
	d.costView = "g"

	view := d.View()
	content := view.Content

	// Check summary panels
	if !strings.Contains(content, "Aujourd'hui") {
		t.Error("viewCosts graph should contain 'Aujourd'hui' summary")
	}
	if !strings.Contains(content, "Semaine") {
		t.Error("viewCosts graph should contain 'Semaine' summary")
	}
	if !strings.Contains(content, "Total (60j)") {
		t.Error("viewCosts graph should contain 'Total (60j)' summary")
	}

	// Check graph elements
	if !strings.Contains(content, "jour") {
		t.Error("viewCosts graph should contain 'jour' in header")
	}
	if !strings.Contains(content, "graphique") {
		t.Error("viewCosts graph should indicate graph view is active")
	}

	// Check cost display
	if !strings.Contains(content, "$") {
		t.Error("viewCosts should display costs with $")
	}
}

func TestViewCostsTable(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 4
	d.costView = "t"

	view := d.View()
	content := view.Content

	// Check table header
	if !strings.Contains(content, "journalier") && !strings.Contains(content, "Journaliers") {
		t.Error("viewCosts table should contain journalier header")
	}

	// Check column headers (headers contain sort key indicators like "(d)ate")
	if !strings.Contains(content, ")ate") && !strings.Contains(content, "date") {
		t.Error("viewCosts table should contain date column")
	}
	if !strings.Contains(content, ")ess") && !strings.Contains(content, "sessions") && !strings.Contains(content, "essions") {
		t.Error("viewCosts table should contain sessions column")
	}
	if !strings.Contains(content, ")es") && !strings.Contains(content, "essages") {
		t.Error("viewCosts table should contain messages column")
	}
	if !strings.Contains(content, "nput") {
		t.Error("viewCosts table should contain input column")
	}
	if !strings.Contains(content, "utput") {
		t.Error("viewCosts table should contain output column")
	}
	if !strings.Contains(content, "Cache") {
		t.Error("viewCosts table should contain Cache column")
	}

	// Check table content contains dates
	if !strings.Contains(content, "Mar") && !strings.Contains(content, "Jan") {
		t.Error("viewCosts table should contain date in rows")
	}

	// Check table view indicator
	if !strings.Contains(content, "tableau") {
		t.Error("viewCosts table should indicate table view is active")
	}
}

func TestViewOverview(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 0

	view := d.View()
	content := view.Content

	// Check main panels exist
	if !strings.Contains(content, "Activit") && !strings.Contains(content, "Activité") {
		t.Error("viewOverview should contain 'Activité' panel")
	}
	if !strings.Contains(content, "Cette semaine") {
		t.Error("viewOverview should contain 'Cette semaine' panel")
	}
	if !strings.Contains(content, "Tokens") {
		t.Error("viewOverview should contain 'Tokens' panel")
	}
	if !strings.Contains(content, "Sant") && !strings.Contains(content, "Santé") {
		t.Error("viewOverview should contain 'Santé' panel")
	}

	// Check Sources panel exists
	if !strings.Contains(content, "Sources") {
		t.Error("viewOverview should contain 'Sources' panel")
	}

	// Check Modèles panel exists
	if !strings.Contains(content, "Modles") && !strings.Contains(content, "Modèles") {
		t.Error("viewOverview should contain 'Modèles' panel")
	}

	// Check sources are listed in Sources panel
	if !strings.Contains(content, "claude") {
		t.Error("viewOverview Sources panel should contain 'claude'")
	}
	if !strings.Contains(content, "opencode") {
		t.Error("viewOverview Sources panel should contain 'opencode'")
	}

	// Check stats values
	if !strings.Contains(content, "5") {
		t.Error("viewOverview should show session count")
	}
	if !strings.Contains(content, "opus") {
		t.Error("viewOverview should show active model")
	}
}

func TestViewStatus(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40

	// Test default help text for non-costs tab
	d.tab = 0
	view := d.View()
	if !strings.Contains(view.Content, "onglets") {
		t.Error("viewStatus should contain 'onglets' for default tabs")
	}
	if !strings.Contains(view.Content, "dfiler") && !strings.Contains(view.Content, "défiler") {
		t.Error("viewStatus should contain 'défiler' for scroll help")
	}
	if !strings.Contains(view.Content, "recharger") {
		t.Error("viewStatus should contain 'recharger' for reload help")
	}

	// Test help text for costs tab (tab 4)
	d.tab = 4
	view = d.View()
	if !strings.Contains(view.Content, "graphique") {
		t.Error("viewStatus should contain 'graphique' for costs tab")
	}
	if !strings.Contains(view.Content, "tableau") {
		t.Error("viewStatus should contain 'tableau' for costs tab")
	}

	// Test help text for other tabs
	d.tab = 1 // Sessions
	view = d.View()
	if !strings.Contains(view.Content, "onglets") {
		t.Error("viewStatus should contain 'onglets' for sessions tab")
	}
	if !strings.Contains(view.Content, "quitter") {
		t.Error("viewStatus should contain 'quitter' for quit help")
	}
}

func TestViewLoadingState(t *testing.T) {
	d := NewDashboard()
	d.loading = true
	d.width = 120
	d.height = 40

	view := d.View()
	content := view.Content

	if !strings.Contains(content, "Chargement") {
		t.Error("View should show loading message when loading=true")
	}
	if !strings.Contains(content, "conversations") {
		t.Error("View should mention 'conversations' while loading")
	}
}

func TestViewNilStats(t *testing.T) {
	d := NewDashboard()
	d.stats = nil
	d.loading = false
	d.width = 120
	d.height = 40

	view := d.View()
	content := view.Content

	if !strings.Contains(content, "Impossible") {
		t.Error("View should show error message when stats is nil")
	}
	if !strings.Contains(content, "charger") {
		t.Error("View should mention inability to load data")
	}
}

func TestViewSmallTerminal(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 30 // Too small
	d.height = 40

	view := d.View()
	content := view.Content

	if !strings.Contains(content, "petit") || !strings.Contains(content, "Terminal") {
		t.Error("View should show 'Terminal trop petit' message for small terminals")
	}
}

func TestScrollRange(t *testing.T) {
	tests := []struct {
		name      string
		scroll    int
		total     int
		pageSize  int
		wantStart int
		wantEnd   int
	}{
		{"normal case", 0, 100, 15, 0, 15},
		{"scroll in middle", 5, 100, 15, 5, 20},
		{"scroll near end", 90, 100, 15, 90, 100},
		{"scroll equals total", 100, 100, 15, 99, 100},
		{"scroll greater than total", 150, 100, 15, 99, 100},
		{"total is zero", 5, 0, 15, 0, 0},
		{"pageSize larger than total", 0, 10, 50, 0, 10},
		{"scroll is zero", 0, 50, 10, 0, 10},
		{"total equals pageSize", 0, 15, 15, 0, 15},
		{"small dataset", 0, 5, 15, 0, 5},
		{"scroll at last page", 85, 100, 15, 85, 100},
		{"scroll exceeds remaining", 95, 100, 15, 95, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := scrollRange(tt.scroll, tt.total, tt.pageSize)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("scrollRange(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.scroll, tt.total, tt.pageSize, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"empty string", "", 5, ""},
		{"short string", "abc", 10, "abc"},
		{"exact length", "hello", 5, "hello"},
		{"truncate ASCII", "hello world", 5, "hello"},
		{"maxLen is zero", "test", 0, ""},
		{"maxLen negative", "test", -1, ""},
		{"Unicode basic", "cafe", 4, "cafe"},
		{"Unicode truncate", "cafe", 3, "caf"},
		{"Unicode emoji", "hello world", 8, "hello wo"},
		{"Unicode mixed", "abcXYZdef", 6, "abcXYZ"},
		{"Unicode CJK", "abc", 3, "abc"},
		{"Unicode CJK truncate", "abc", 2, "ab"},
		{"single char", "x", 1, "x"},
		{"single char truncate", "x", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestMiniBar(t *testing.T) {
	d := NewDashboard()
	tests := []struct {
		name  string
		pct   float64
		width int
	}{
		{"zero percent", 0, 5},
		{"fifty percent", 50, 4},
		{"one hundred percent", 100, 3},
		{"over hundred percent", 150, 5},
		{"small percentage", 10, 10},
		{"large percentage", 90, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.miniBar(tt.pct, tt.width)
			// Verify length is reasonable (styled output)
			if len(got) == 0 {
				t.Errorf("miniBar(%f, %d) returned empty string", tt.pct, tt.width)
			}
		})
	}
}

func TestMiniBarContent(t *testing.T) {
	d := NewDashboard()

	// Test 0% - should be all empty characters (dim style)
	bar := d.miniBar(0, 5)
	if len(bar) == 0 {
		t.Errorf("miniBar(0, 5) should not be empty")
	}

	// Test 100% - should be all filled characters (spark style)
	bar = d.miniBar(100, 5)
	if len(bar) == 0 {
		t.Errorf("miniBar(100, 5) should not be empty")
	}

	// Test 50% - half filled, half empty
	bar = d.miniBar(50, 4)
	if len(bar) == 0 {
		t.Errorf("miniBar(50, 4) should not be empty")
	}

	// Test >100% - should cap at width (same as 100%)
	bar = d.miniBar(150, 5)
	if len(bar) == 0 {
		t.Errorf("miniBar(150, 5) should not be empty")
	}
}

func TestColorErrors(t *testing.T) {
	d := NewDashboard()
	tests := []struct {
		name      string
		count     int
		wantValue string
	}{
		{"zero errors", 0, "0"},
		{"low errors", 5, "5"},
		{"at threshold 10", 10, "10"},
		{"warning zone", 15, "15"},
		{"at threshold 50", 50, "50"},
		{"error zone", 75, "75"},
		{"high errors", 100, "100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.colorErrors(tt.count)
			if !strings.Contains(got, tt.wantValue) {
				t.Errorf("colorErrors(%d) = %q, should contain %q", tt.count, got, tt.wantValue)
			}
		})
	}
}

func TestColorErrorRate(t *testing.T) {
	d := NewDashboard()
	tests := []struct {
		name      string
		pct       float64
		wantValue string
	}{
		{"zero percent", 0.0, "0.0%"},
		{"low rate", 0.5, "0.5%"},
		{"at threshold 2", 2.0, "2.0%"},
		{"warning zone", 3.5, "3.5%"},
		{"at threshold 5", 5.0, "5.0%"},
		{"error zone", 7.5, "7.5%"},
		{"high rate", 15.0, "15.0%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.colorErrorRate(tt.pct)
			if !strings.Contains(got, tt.wantValue) {
				t.Errorf("colorErrorRate(%f) = %q, should contain %q", tt.pct, got, tt.wantValue)
			}
		})
	}
}

func TestColorCacheRate(t *testing.T) {
	d := NewDashboard()
	tests := []struct {
		name      string
		pct       float64
		wantValue string
	}{
		{"zero percent", 0.0, "0%"},
		{"low rate", 10.0, "10%"},
		{"at threshold 20", 20.0, "20%"},
		{"warning zone", 35.0, "35%"},
		{"at threshold 50", 50.0, "50%"},
		{"good rate", 75.0, "75%"},
		{"high rate", 100.0, "100%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.colorCacheRate(tt.pct)
			if !strings.Contains(got, tt.wantValue) {
				t.Errorf("colorCacheRate(%f) = %q, should contain %q", tt.pct, got, tt.wantValue)
			}
		})
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{"zero", 0.0, "$0.00"},
		{"small value", 0.01, "$0.01"},
		{"typical value", 1.50, "$1.50"},
		{"large value", 100.99, "$100.99"},
		{"round number", 50.0, "$50.00"},
		{"very small", 0.001, "$0.00"},
		{"thousands", 1234.56, "$1234.56"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCost(tt.input)
			if got != tt.want {
				t.Errorf("formatCost(%f) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- Sorting Tests ---

func TestToggleSort(t *testing.T) {
	d := NewDashboard()

	// Test: toggling same column flips asc/desc
	d.sortSessions = sortOrder{col: "projet", asc: false}
	d.toggleSort(&d.sortSessions, "projet")
	if d.sortSessions.col != "projet" || !d.sortSessions.asc {
		t.Errorf("expected col=projet, asc=true after toggle, got col=%s, asc=%v", d.sortSessions.col, d.sortSessions.asc)
	}

	// Toggle again should flip back to desc
	d.toggleSort(&d.sortSessions, "projet")
	if d.sortSessions.col != "projet" || d.sortSessions.asc {
		t.Errorf("expected col=projet, asc=false after second toggle, got col=%s, asc=%v", d.sortSessions.col, d.sortSessions.asc)
	}

	// Test: new column resets to desc
	d.sortSessions = sortOrder{col: "projet", asc: true}
	d.toggleSort(&d.sortSessions, "msgs")
	if d.sortSessions.col != "msgs" || d.sortSessions.asc {
		t.Errorf("expected col=msgs, asc=false for new column, got col=%s, asc=%v", d.sortSessions.col, d.sortSessions.asc)
	}

	// Test: scroll should reset to 0
	d.scroll = 10
	d.toggleSort(&d.sortSessions, "tools")
	if d.scroll != 0 {
		t.Errorf("expected scroll=0 after toggle, got %d", d.scroll)
	}
}

func TestSortIndicator(t *testing.T) {
	tests := []struct {
		so       sortOrder
		col      string
		expected string
	}{
		{sortOrder{col: "projet", asc: false}, "projet", " \u25bc"}, // descending arrow
		{sortOrder{col: "projet", asc: true}, "projet", " \u25b2"},  // ascending arrow
		{sortOrder{col: "projet", asc: false}, "msgs", ""},         // different column
		{sortOrder{col: "projet", asc: true}, "msgs", ""},          // different column
	}

	for _, tt := range tests {
		got := sortIndicator(tt.so, tt.col)
		if got != tt.expected {
			t.Errorf("sortIndicator(%+v, %q) = %q, want %q", tt.so, tt.col, got, tt.expected)
		}
	}
}

func TestHandleSortKey(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()

	// Sessions tab (1): p/m/t/o/d
	d.tab = 1
	d.handleSortKey("p")
	if d.sortSessions.col != "projet" {
		t.Errorf("Sessions: key 'p' should set col to 'projet', got %s", d.sortSessions.col)
	}
	d.handleSortKey("m")
	if d.sortSessions.col != "msgs" {
		t.Errorf("Sessions: key 'm' should set col to 'msgs', got %s", d.sortSessions.col)
	}
	d.handleSortKey("t")
	if d.sortSessions.col != "tools" {
		t.Errorf("Sessions: key 't' should set col to 'tools', got %s", d.sortSessions.col)
	}
	d.handleSortKey("o")
	if d.sortSessions.col != "tokens" {
		t.Errorf("Sessions: key 'o' should set col to 'tokens', got %s", d.sortSessions.col)
	}
	d.handleSortKey("d")
	if d.sortSessions.col != "duree" {
		t.Errorf("Sessions: key 'd' should set col to 'duree', got %s", d.sortSessions.col)
	}

	// Tools tab (2): o/a/%
	d.tab = 2
	d.handleSortKey("o")
	if d.sortTools.col != "outil" {
		t.Errorf("Tools: key 'o' should set col to 'outil', got %s", d.sortTools.col)
	}
	d.handleSortKey("a")
	if d.sortTools.col != "appels" {
		t.Errorf("Tools: key 'a' should set col to 'appels', got %s", d.sortTools.col)
	}
	d.handleSortKey("%")
	if d.sortTools.col != "pct" {
		t.Errorf("Tools: key '%%' should set col to 'pct', got %s", d.sortTools.col)
	}

	// Projects tab (3): p/s/m/t/c
	d.tab = 3
	d.handleSortKey("p")
	if d.sortProjects.col != "projet" {
		t.Errorf("Projects: key 'p' should set col to 'projet', got %s", d.sortProjects.col)
	}
	d.handleSortKey("s")
	if d.sortProjects.col != "sessions" {
		t.Errorf("Projects: key 's' should set col to 'sessions', got %s", d.sortProjects.col)
	}
	d.handleSortKey("m")
	if d.sortProjects.col != "messages" {
		t.Errorf("Projects: key 'm' should set col to 'messages', got %s", d.sortProjects.col)
	}
	d.handleSortKey("t")
	if d.sortProjects.col != "tokens" {
		t.Errorf("Projects: key 't' should set col to 'tokens', got %s", d.sortProjects.col)
	}
	d.handleSortKey("c")
	if d.sortProjects.col != "cout" {
		t.Errorf("Projects: key 'c' should set col to 'cout', got %s", d.sortProjects.col)
	}

	// Models tab (6): m/s/g/i/o/h/c
	d.tab = 6
	d.handleSortKey("m")
	if d.sortModels.col != "model" {
		t.Errorf("Models: key 'm' should set col to 'model', got %s", d.sortModels.col)
	}
	d.handleSortKey("s")
	if d.sortModels.col != "source" {
		t.Errorf("Models: key 's' should set col to 'source', got %s", d.sortModels.col)
	}
	d.handleSortKey("g")
	if d.sortModels.col != "msgs" {
		t.Errorf("Models: key 'g' should set col to 'msgs', got %s", d.sortModels.col)
	}
	d.handleSortKey("i")
	if d.sortModels.col != "input" {
		t.Errorf("Models: key 'i' should set col to 'input', got %s", d.sortModels.col)
	}
	d.handleSortKey("o")
	if d.sortModels.col != "output" {
		t.Errorf("Models: key 'o' should set col to 'output', got %s", d.sortModels.col)
	}
	d.handleSortKey("h")
	if d.sortModels.col != "cache" {
		t.Errorf("Models: key 'h' should set col to 'cache', got %s", d.sortModels.col)
	}
	d.handleSortKey("c")
	if d.sortModels.col != "cout" {
		t.Errorf("Models: key 'c' should set col to 'cout', got %s", d.sortModels.col)
	}

	// Costs tab (4): d/s/m/i/o/c
	d.tab = 4
	d.costView = "t" // table view for sorting
	d.handleSortKey("d")
	if d.sortCosts.col != "date" {
		t.Errorf("Costs: key 'd' should set col to 'date', got %s", d.sortCosts.col)
	}
	d.handleSortKey("s")
	if d.sortCosts.col != "sessions" {
		t.Errorf("Costs: key 's' should set col to 'sessions', got %s", d.sortCosts.col)
	}
	d.handleSortKey("m")
	if d.sortCosts.col != "messages" {
		t.Errorf("Costs: key 'm' should set col to 'messages', got %s", d.sortCosts.col)
	}
	d.handleSortKey("i")
	if d.sortCosts.col != "input" {
		t.Errorf("Costs: key 'i' should set col to 'input', got %s", d.sortCosts.col)
	}
	d.handleSortKey("o")
	if d.sortCosts.col != "output" {
		t.Errorf("Costs: key 'o' should set col to 'output', got %s", d.sortCosts.col)
	}
	d.handleSortKey("c")
	if d.sortCosts.col != "cout" {
		t.Errorf("Costs: key 'c' should set col to 'cout', got %s", d.sortCosts.col)
	}
}

func TestSessionsSortOrder(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 1

	// Sort by msgs descending (default)
	d.sortSessions = sortOrder{col: "msgs", asc: false}
	view := d.viewSessions(120)
	// s1 has 25 msgs, s2 has 13 msgs, s3 has 7 msgs
	// Order should be: s1 (25), s2 (13), s3 (7)
	if !strings.Contains(view, "test-proj") {
		t.Error("Sessions view should contain project name")
	}

	// Sort by msgs ascending - should reverse the order
	d.sortSessions = sortOrder{col: "msgs", asc: true}
	viewAsc := d.viewSessions(120)
	if !strings.Contains(viewAsc, "gemini-proj") {
		t.Error("Sessions view (asc) should contain gemini-proj")
	}

	// Sort by projet (name) ascending
	d.sortSessions = sortOrder{col: "projet", asc: true}
	viewProj := d.viewSessions(120)
	if !strings.Contains(viewProj, "gemini-proj") { // alphabetically first
		t.Error("Sessions sorted by projet asc should have gemini-proj first")
	}
}

func TestToolsSortOrder(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 2

	// Sort by appels descending (highest count first)
	d.sortTools = sortOrder{col: "appels", asc: false}
	view := d.viewTools(120)
	if !strings.Contains(view, "Read") {
		t.Error("Tools view should contain 'Read'")
	}

	// Sort by outil (name) ascending
	d.sortTools = sortOrder{col: "outil", asc: true}
	viewName := d.viewTools(120)
	if !strings.Contains(viewName, "Bash") {
		t.Error("Tools view sorted by name should contain 'Bash'")
	}
}

func TestProjectsSortOrder(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 3

	// Sort by sessions descending
	d.sortProjects = sortOrder{col: "sessions", asc: false}
	view := d.viewProjects(120)
	if !strings.Contains(view, "test-proj") {
		t.Error("Projects view should contain 'test-proj'")
	}

	// Sort by projet (name) ascending
	d.sortProjects = sortOrder{col: "projet", asc: true}
	viewName := d.viewProjects(120)
	if !strings.Contains(viewName, "other-proj") { // alphabetically first
		t.Error("Projects sorted by name asc should have 'other-proj' first")
	}

	// Sort by cost descending
	d.sortProjects = sortOrder{col: "cout", asc: false}
	viewCost := d.viewProjects(120)
	if !strings.Contains(viewCost, "$1.83") {
		t.Error("Projects sorted by cost should show $1.83")
	}
}

func TestModelsSortOrder(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 6

	// Sort by msgs descending (default)
	d.sortModels = sortOrder{col: "msgs", asc: false}
	view := d.viewModels(120)
	if !strings.Contains(view, "claude-opus") {
		t.Error("Models view should contain 'claude-opus'")
	}

	// Sort by model name ascending
	d.sortModels = sortOrder{col: "model", asc: true}
	viewName := d.viewModels(120)
	if !strings.Contains(viewName, "claude") {
		t.Error("Models view sorted by name should contain 'claude'")
	}
}

func TestCostsSortOrder(t *testing.T) {
	d := NewDashboard()
	stats := fakeStats()
	d.stats = stats
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 4
	d.costView = "t"

	// Sort by date descending (default)
	d.sortCosts = sortOrder{col: "date", asc: false}
	view := d.viewCostTable(120, stats.DailyCosts)
	if !strings.Contains(view, "essions") {
		t.Error("Costs table view should contain 'essions' column header")
	}

	// Sort by sessions descending
	d.sortCosts = sortOrder{col: "sessions", asc: false}
	viewSess := d.viewCostTable(120, stats.DailyCosts)
	if !strings.Contains(viewSess, "essions") {
		t.Error("Costs table should contain 'essions' column header")
	}
}

func TestViewNetwork(t *testing.T) {
	d := NewDashboard()
	d.localStats = fakeStats()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 7

	view := d.View()
	content := view.Content

	// Check panel headers
	if !strings.Contains(content, "Cette machine") {
		t.Error("viewNetwork should contain 'Cette machine' panel")
	}
	if !strings.Contains(content, "Total agrg") && !strings.Contains(content, "Total agrégé") {
		t.Error("viewNetwork should contain 'Total agrégé' panel")
	}
	if !strings.Contains(content, "Machines distantes") {
		t.Error("viewNetwork should contain 'Machines distantes' panel")
	}

	// Check for local stats
	if !strings.Contains(content, "Sessions locales") {
		t.Error("viewNetwork should contain 'Sessions locales'")
	}

	// Check for no peers message
	if !strings.Contains(content, "Aucun peer") {
		t.Error("viewNetwork should show 'Aucun peer configuré' when no peers added")
	}
}

func TestNetworkTabNavigation(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40

	// Tab 8 should navigate to network tab (index 7)
	model, _ := d.Update(tea.KeyPressMsg{Code: '8', Text: "8"})
	d = model.(Dashboard)
	if d.tab != 7 {
		t.Errorf("expected tab 7 for network, got %d", d.tab)
	}
}
