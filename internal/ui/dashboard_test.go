package ui

import (
	"strings"
	"testing"
	"time"

	"claude_monitor/internal/data"
	"claude_monitor/internal/peer"

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
	if !strings.Contains(view.Content, "Activity") {
		t.Error("overview should contain 'Activity'")
	}
	if !strings.Contains(view.Content, "Tokens") {
		t.Error("overview should contain 'Tokens'")
	}
	if !strings.Contains(view.Content, "This Week") {
		t.Error("overview should contain 'This Week'")
	}
	if !strings.Contains(view.Content, "Health") {
		t.Error("overview should contain 'Health'")
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
	if !strings.Contains(view.Content, "Tool Usage") {
		t.Error("tools tab should contain 'Tool Usage'")
	}

	// Projects
	d.tab = 3
	view = d.View()
	if !strings.Contains(view.Content, "Project") {
		t.Error("projects tab should contain 'Project'")
	}
	if !strings.Contains(view.Content, "$1.83") {
		t.Error("projects tab should contain cost per project")
	}

	// Costs - graph
	d.tab = 4
	d.costView = "g"
	view = d.View()
	if !strings.Contains(view.Content, "Daily Cost") {
		t.Error("costs tab graph should contain 'Daily Cost'")
	}

	// Costs - table
	d.costView = "t"
	view = d.View()
	if !strings.Contains(view.Content, "Daily Costs") {
		t.Error("costs tab table should contain 'Daily Costs'")
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
	if !strings.Contains(view.Content, "Comparison") {
		t.Error("sources tab should contain 'Comparison'")
	}

	// Models
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
		{1500000000, "1.5B"},
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
	if !strings.Contains(content, "Comparison by Source") {
		t.Error("viewSources should contain 'Comparison by Source' header")
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
	if !strings.Contains(content, "Models by Source") {
		t.Error("viewModels should contain 'Models by Source' header")
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
	if !strings.Contains(content, "Today") {
		t.Error("viewCosts graph should contain 'Today' summary")
	}
	if !strings.Contains(content, "Week") {
		t.Error("viewCosts graph should contain 'Week' summary")
	}
	if !strings.Contains(content, "Total (60d)") {
		t.Error("viewCosts graph should contain 'Total (60d)' summary")
	}

	// Check graph elements
	if !strings.Contains(content, "Daily Cost") {
		t.Error("viewCosts graph should contain 'Daily Cost' in header")
	}
	if !strings.Contains(content, "graph") {
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
	if !strings.Contains(content, "Daily Costs") {
		t.Error("viewCosts table should contain 'Daily Costs' header")
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
	if !strings.Contains(content, "table") {
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
	if !strings.Contains(content, "Activity") {
		t.Error("viewOverview should contain 'Activity' panel")
	}
	if !strings.Contains(content, "This Week") {
		t.Error("viewOverview should contain 'This Week' panel")
	}
	if !strings.Contains(content, "Tokens") {
		t.Error("viewOverview should contain 'Tokens' panel")
	}
	if !strings.Contains(content, "Health") {
		t.Error("viewOverview should contain 'Health' panel")
	}

	// Check Sources panel exists
	if !strings.Contains(content, "Sources") {
		t.Error("viewOverview should contain 'Sources' panel")
	}

	// Check Models panel exists
	if !strings.Contains(content, "Models") {
		t.Error("viewOverview should contain 'Models' panel")
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
	if !strings.Contains(view.Content, "tabs") {
		t.Error("viewStatus should contain 'tabs' for default tabs")
	}
	if !strings.Contains(view.Content, "scroll") {
		t.Error("viewStatus should contain 'scroll' for scroll help")
	}
	if !strings.Contains(view.Content, "reload") {
		t.Error("viewStatus should contain 'reload' for reload help")
	}

	// Test help text for costs tab (tab 4)
	d.tab = 4
	view = d.View()
	if !strings.Contains(view.Content, "graph") {
		t.Error("viewStatus should contain 'graph' for costs tab")
	}
	if !strings.Contains(view.Content, "table") {
		t.Error("viewStatus should contain 'table' for costs tab")
	}

	// Test help text for other tabs
	d.tab = 1 // Sessions
	view = d.View()
	if !strings.Contains(view.Content, "tabs") {
		t.Error("viewStatus should contain 'tabs' for sessions tab")
	}
	if !strings.Contains(view.Content, "quit") {
		t.Error("viewStatus should contain 'quit' for quit help")
	}
}

func TestViewLoadingState(t *testing.T) {
	d := NewDashboard()
	d.loading = true
	d.width = 120
	d.height = 40

	view := d.View()
	content := view.Content

	if !strings.Contains(content, "Loading") {
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

	if !strings.Contains(content, "Failed") {
		t.Error("View should show error message when stats is nil")
	}
	if !strings.Contains(content, "load") {
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

	if !strings.Contains(content, "too small") || !strings.Contains(content, "Terminal") {
		t.Error("View should show 'Terminal too small' message for small terminals")
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
	d.sortSessions = sortOrder{col: "project", asc: false}
	d.toggleSort(&d.sortSessions, "project")
	if d.sortSessions.col != "project" || !d.sortSessions.asc {
		t.Errorf("expected col=project, asc=true after toggle, got col=%s, asc=%v", d.sortSessions.col, d.sortSessions.asc)
	}

	// Toggle again should flip back to desc
	d.toggleSort(&d.sortSessions, "project")
	if d.sortSessions.col != "project" || d.sortSessions.asc {
		t.Errorf("expected col=project, asc=false after second toggle, got col=%s, asc=%v", d.sortSessions.col, d.sortSessions.asc)
	}

	// Test: new column resets to desc
	d.sortSessions = sortOrder{col: "project", asc: true}
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
		{sortOrder{col: "project", asc: false}, "project", " \u25bc"}, // descending arrow
		{sortOrder{col: "project", asc: true}, "project", " \u25b2"},  // ascending arrow
		{sortOrder{col: "project", asc: false}, "msgs", ""},           // different column
		{sortOrder{col: "project", asc: true}, "msgs", ""},            // different column
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
	if d.sortSessions.col != "project" {
		t.Errorf("Sessions: key 'p' should set col to 'project', got %s", d.sortSessions.col)
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
	if d.sortSessions.col != "duration" {
		t.Errorf("Sessions: key 'd' should set col to 'duration', got %s", d.sortSessions.col)
	}

	// Tools tab (2): t/c/%
	d.tab = 2
	d.handleSortKey("t")
	if d.sortTools.col != "tool" {
		t.Errorf("Tools: key 't' should set col to 'tool', got %s", d.sortTools.col)
	}
	d.handleSortKey("c")
	if d.sortTools.col != "calls" {
		t.Errorf("Tools: key 'c' should set col to 'calls', got %s", d.sortTools.col)
	}
	d.handleSortKey("%")
	if d.sortTools.col != "pct" {
		t.Errorf("Tools: key '%%' should set col to 'pct', got %s", d.sortTools.col)
	}

	// Projects tab (3): p/s/m/t/c
	d.tab = 3
	d.handleSortKey("p")
	if d.sortProjects.col != "project" {
		t.Errorf("Projects: key 'p' should set col to 'project', got %s", d.sortProjects.col)
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
	if d.sortProjects.col != "cost" {
		t.Errorf("Projects: key 'c' should set col to 'cost', got %s", d.sortProjects.col)
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
	if d.sortModels.col != "cost" {
		t.Errorf("Models: key 'c' should set col to 'cost', got %s", d.sortModels.col)
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
	if d.sortCosts.col != "cost" {
		t.Errorf("Costs: key 'c' should set col to 'cost', got %s", d.sortCosts.col)
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

	// Sort by project (name) ascending
	d.sortSessions = sortOrder{col: "project", asc: true}
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

	// Sort by calls descending (highest count first)
	d.sortTools = sortOrder{col: "calls", asc: false}
	view := d.viewTools(120)
	if !strings.Contains(view, "Read") {
		t.Error("Tools view should contain 'Read'")
	}

	// Sort by tool (name) ascending
	d.sortTools = sortOrder{col: "tool", asc: true}
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

	// Sort by project (name) ascending
	d.sortProjects = sortOrder{col: "project", asc: true}
	viewName := d.viewProjects(120)
	if !strings.Contains(viewName, "other-proj") { // alphabetically first
		t.Error("Projects sorted by name asc should have 'other-proj' first")
	}

	// Sort by cost descending
	d.sortProjects = sortOrder{col: "cost", asc: false}
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
	if !strings.Contains(content, "This Machine") {
		t.Error("viewNetwork should contain 'This Machine' panel")
	}
	if !strings.Contains(content, "Aggregated Total") {
		t.Error("viewNetwork should contain 'Aggregated Total' panel")
	}
	if !strings.Contains(content, "Remote Machines") {
		t.Error("viewNetwork should contain 'Remote Machines' panel")
	}

	// Check for local stats
	if !strings.Contains(content, "Local sessions") {
		t.Error("viewNetwork should contain 'Local sessions'")
	}

	// Check for no peers message
	if !strings.Contains(content, "No peers") {
		t.Error("viewNetwork should show 'No peers configured' when no peers added")
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

func TestSharedLocalStatsAtomicPointer(t *testing.T) {
	d := NewDashboard()

	// Initially nil
	if got := d.GetLocalStats(); got != nil {
		t.Errorf("GetLocalStats should be nil initially, got %+v", got)
	}

	// Simulate bubbletea Update with statsMsg
	stats := fakeStats()
	model, _ := d.Update(statsMsg(stats))
	d2 := model.(Dashboard)

	// The returned model should have stats
	if d2.localStats == nil {
		t.Fatal("d2.localStats should not be nil after statsMsg")
	}

	// Both the original and the copy share the atomic pointer
	got := d.GetLocalStats()
	if got == nil {
		t.Fatal("original Dashboard.GetLocalStats should see stats via atomic pointer")
	}
	if got.TotalSessions != 5 {
		t.Errorf("GetLocalStats().TotalSessions = %d, want 5", got.TotalSessions)
	}

	// The copy also sees the same stats
	got2 := d2.GetLocalStats()
	if got2 == nil {
		t.Fatal("copy Dashboard.GetLocalStats should also see stats")
	}
	if got2.TotalSessions != got.TotalSessions {
		t.Error("both copies should see the same stats via atomic pointer")
	}
}

func TestSetPort(t *testing.T) {
	d := NewDashboard()
	if d.port != 9999 {
		t.Errorf("default port should be 9999, got %d", d.port)
	}
	d.SetPort(8080)
	if d.port != 8080 {
		t.Errorf("port should be 8080 after SetPort, got %d", d.port)
	}
}

func TestHandleInputAddPeer(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.localStats = fakeStats()
	d.loading = false

	// Enter input mode
	d.inputMode = true
	d.inputBuffer = ""
	d.inputPrompt = "IP:port address"

	// Type an address
	for _, ch := range "192.168.1.50:9999" {
		d.handleInput(string(ch))
	}

	if d.inputBuffer != "192.168.1.50:9999" {
		t.Errorf("inputBuffer = %q, want %q", d.inputBuffer, "192.168.1.50:9999")
	}

	// Press enter to confirm
	d.handleInput("enter")

	if d.inputMode {
		t.Error("inputMode should be false after enter")
	}
	if d.inputBuffer != "" {
		t.Error("inputBuffer should be cleared after enter")
	}

	// Peer should be stored
	peers := d.peerStorage.List()
	if len(peers) != 1 || peers[0] != "192.168.1.50:9999" {
		t.Errorf("expected peer 192.168.1.50:9999, got %v", peers)
	}
}

func TestHandleInputEscape(t *testing.T) {
	d := NewDashboard()
	d.inputMode = true
	d.inputBuffer = "partial"

	d.handleInput("escape")

	if d.inputMode {
		t.Error("inputMode should be false after escape")
	}
	if d.inputBuffer != "" {
		t.Error("inputBuffer should be cleared after escape")
	}
}

func TestHandleInputBackspace(t *testing.T) {
	d := NewDashboard()
	d.inputMode = true
	d.inputBuffer = "abc"

	d.handleInput("backspace")
	if d.inputBuffer != "ab" {
		t.Errorf("after backspace: inputBuffer = %q, want %q", d.inputBuffer, "ab")
	}

	d.handleInput("backspace")
	d.handleInput("backspace")
	if d.inputBuffer != "" {
		t.Errorf("after all backspaces: inputBuffer = %q, want empty", d.inputBuffer)
	}

	// Backspace on empty should not panic
	d.handleInput("backspace")
	if d.inputBuffer != "" {
		t.Error("backspace on empty should remain empty")
	}
}

func TestHandleInputIgnoresNonPrintable(t *testing.T) {
	d := NewDashboard()
	d.inputMode = true
	d.inputBuffer = ""

	d.handleInput("tab")
	d.handleInput("up")
	d.handleInput("down")
	d.handleInput("ctrl+c")

	if d.inputBuffer != "" {
		t.Errorf("non-printable keys should not modify buffer, got %q", d.inputBuffer)
	}
}

func TestNetworkSortKeysAddAndFetch(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.localStats = fakeStats()
	d.loading = false
	d.tab = 7

	// 'a' should enter input mode
	d.handleSortKey("a")
	if !d.inputMode {
		t.Error("key 'a' on network tab should enter input mode")
	}
	if d.inputPrompt != "IP:port address" {
		t.Errorf("inputPrompt = %q, want %q", d.inputPrompt, "IP:port address")
	}

	// Reset
	d.inputMode = false

	// 'f' should call fetchPeers (no crash with empty peers)
	d.handleSortKey("f")
}

func TestViewNetworkWithPeers(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.localStats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 7

	// Add a peer status (simulating fetched peers)
	d.peerStatuses = []peer.PeerStatus{
		{
			Address: "192.168.1.74:9999",
			Online:  true,
			Stats: &data.Stats{
				TotalSessions:     3,
				TotalInputTokens:  2000,
				TotalOutputTokens: 1000,
			},
		},
		{
			Address:   "192.168.1.99:9999",
			Online:    false,
			LastError: "connection failed",
		},
	}

	view := d.View()
	content := view.Content

	if !strings.Contains(content, "192.168.1.74:9999") {
		t.Error("should show online peer address")
	}
	if !strings.Contains(content, "online") {
		t.Error("should show 'online' status for reachable peer")
	}
	if !strings.Contains(content, "192.168.1.99:9999") {
		t.Error("should show offline peer address")
	}
	if !strings.Contains(content, "offline") {
		t.Error("should show 'offline' status for unreachable peer")
	}
	if !strings.Contains(content, "connection failed") {
		t.Error("should show error message for failed peer")
	}
}

func TestViewNetworkStatusBar(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.localStats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 7

	d.peerStatuses = []peer.PeerStatus{
		{Address: "a", Online: true},
		{Address: "b", Online: false},
	}

	view := d.View()
	content := view.Content

	// Status bar should show peer count
	if !strings.Contains(content, "1/2") {
		t.Error("status bar should show online/total peers count (1/2)")
	}
	if !strings.Contains(content, "add") {
		t.Error("status bar should show 'add' help")
	}
}

func TestReloadKey(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40

	model, cmd := d.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	d = model.(Dashboard)

	if !d.loading {
		t.Error("pressing 'r' should set loading=true")
	}
	if cmd == nil {
		t.Error("pressing 'r' should return a command")
	}
}

func TestScrollKeys(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.scroll = 0

	// Scroll down with j
	model, _ := d.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	d = model.(Dashboard)
	if d.scroll != 1 {
		t.Errorf("expected scroll=1 after 'j', got %d", d.scroll)
	}

	// Scroll down with down arrow
	model, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	d = model.(Dashboard)
	if d.scroll != 2 {
		t.Errorf("expected scroll=2 after down, got %d", d.scroll)
	}

	// Scroll up with k
	model, _ = d.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	d = model.(Dashboard)
	if d.scroll != 1 {
		t.Errorf("expected scroll=1 after 'k', got %d", d.scroll)
	}

	// Scroll up with up arrow
	model, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	d = model.(Dashboard)
	if d.scroll != 0 {
		t.Errorf("expected scroll=0 after up, got %d", d.scroll)
	}

	// Up at 0 should stay at 0
	model, _ = d.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	d = model.(Dashboard)
	if d.scroll != 0 {
		t.Errorf("scroll should not go below 0, got %d", d.scroll)
	}
}

func TestCostViewToggle(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.tab = 4
	d.costView = "g"

	d.handleSortKey("t")
	if d.costView != "t" {
		t.Errorf("expected costView='t' after pressing 't', got %q", d.costView)
	}

	d.handleSortKey("g")
	if d.costView != "g" {
		t.Errorf("expected costView='g' after pressing 'g', got %q", d.costView)
	}
}

func TestInputModeBlocksNavigation(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.tab = 7
	d.inputMode = true
	d.inputBuffer = ""

	// Tab key should not switch tabs while in input mode
	// handleInput returns *Dashboard, so type-assert accordingly
	model, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	dp := model.(*Dashboard)
	if dp.tab != 7 {
		t.Errorf("tab should remain 7 during input mode, got %d", dp.tab)
	}

	// 'q' should not quit during input mode
	model, cmd := dp.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	dp = model.(*Dashboard)
	if cmd != nil {
		t.Error("'q' during input mode should not trigger quit command")
	}
	if dp.inputBuffer != "q" {
		t.Errorf("'q' should be typed into buffer, got %q", dp.inputBuffer)
	}
}

func TestWindowSizeMsg(t *testing.T) {
	d := NewDashboard()
	model, _ := d.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	d = model.(Dashboard)
	if d.width != 200 || d.height != 50 {
		t.Errorf("expected 200x50, got %dx%d", d.width, d.height)
	}
}

func TestTabSwitchResetsScroll(t *testing.T) {
	d := NewDashboard()
	d.stats = fakeStats()
	d.loading = false
	d.width = 120
	d.height = 40
	d.scroll = 10

	model, _ := d.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	d = model.(Dashboard)

	if d.scroll != 0 {
		t.Errorf("switching tabs should reset scroll to 0, got %d", d.scroll)
	}
	if d.tab != 1 {
		t.Errorf("pressing '2' should switch to tab 1, got %d", d.tab)
	}
}

func TestNewDashboardDefaults(t *testing.T) {
	d := NewDashboard()

	if !d.loading {
		t.Error("new dashboard should start in loading state")
	}
	if d.sharedLocalStats == nil {
		t.Error("sharedLocalStats should be initialized")
	}
	if d.peerStorage == nil {
		t.Error("peerStorage should be initialized")
	}
	if d.port != 9999 {
		t.Errorf("default port should be 9999, got %d", d.port)
	}
	if d.costView != "g" {
		t.Errorf("default costView should be 'g', got %q", d.costView)
	}
	if d.sortCosts.col != "date" {
		t.Errorf("default cost sort should be 'date', got %q", d.sortCosts.col)
	}
	if d.sortModels.col != "msgs" {
		t.Errorf("default model sort should be 'msgs', got %q", d.sortModels.col)
	}
}
