package data

import (
	"testing"
	"time"
)

func TestStatsMerge(t *testing.T) {
	// Create base stats
	base := &Stats{
		Sessions: []Session{
			{ID: "s1", Project: "proj1", UserMessages: 5, AssistantMessages: 3},
		},
		Projects: []ProjectSummary{
			{Name: "proj1", Sessions: 1, Messages: 8},
		},
		ToolUsage: map[string]int{
			"Read": 10,
			"Bash": 5,
		},
		TotalInputTokens:  1000,
		TotalOutputTokens: 500,
		TotalCacheRead:    200,
		TotalMessages:     8,
		TotalToolUses:     15,
		TotalToolErrors:   1,
		TotalSessions:     1,
		ActiveSessions:    1,
		TodaySessions:     1,
		TodayMessages:     8,
		TodayTokens:       1500,
		WeekSessions:      1,
		WeekMessages:      8,
		WeekTokens:        1500,
		TotalCost:         1.50,
		Models: map[string]int{
			"claude-opus-4-6": 3,
		},
		DailyCosts: []DayCost{
			{Date: time.Now(), InputTokens: 1000, OutputTokens: 500, Cost: 1.50},
		},
	}

	// Create remote stats to merge
	remote := &Stats{
		Sessions: []Session{
			{ID: "s2", Project: "proj2", UserMessages: 4, AssistantMessages: 2},
		},
		Projects: []ProjectSummary{
			{Name: "proj2", Sessions: 1, Messages: 6},
		},
		ToolUsage: map[string]int{
			"Read": 8,
			"Edit": 3,
		},
		TotalInputTokens:  2000,
		TotalOutputTokens: 1000,
		TotalCacheRead:    400,
		TotalMessages:     6,
		TotalToolUses:     11,
		TotalToolErrors:   2,
		TotalSessions:     1,
		ActiveSessions:    1,
		TodaySessions:     1,
		TodayMessages:     6,
		TodayTokens:       3000,
		WeekSessions:      1,
		WeekMessages:      6,
		WeekTokens:        3000,
		TotalCost:         3.00,
		Models: map[string]int{
			"claude-opus-4-6": 2,
			"claude-haiku-4-5": 1,
		},
		DailyCosts: []DayCost{
			{Date: time.Now(), InputTokens: 2000, OutputTokens: 1000, Cost: 3.00},
		},
	}

	// Merge
	base.Merge(remote)

	// Verify totals are summed
	if base.TotalInputTokens != 3000 {
		t.Errorf("TotalInputTokens: got %d, want 3000", base.TotalInputTokens)
	}
	if base.TotalOutputTokens != 1500 {
		t.Errorf("TotalOutputTokens: got %d, want 1500", base.TotalOutputTokens)
	}
	if base.TotalCacheRead != 600 {
		t.Errorf("TotalCacheRead: got %d, want 600", base.TotalCacheRead)
	}
	if base.TotalMessages != 14 {
		t.Errorf("TotalMessages: got %d, want 14", base.TotalMessages)
	}
	if base.TotalToolUses != 26 {
		t.Errorf("TotalToolUses: got %d, want 26", base.TotalToolUses)
	}
	if base.TotalToolErrors != 3 {
		t.Errorf("TotalToolErrors: got %d, want 3", base.TotalToolErrors)
	}
	if base.TotalSessions != 2 {
		t.Errorf("TotalSessions: got %d, want 2", base.TotalSessions)
	}
	if base.TotalCost != 4.50 {
		t.Errorf("TotalCost: got %.2f, want 4.50", base.TotalCost)
	}

	// Verify sessions are appended
	if len(base.Sessions) != 2 {
		t.Errorf("Sessions count: got %d, want 2", len(base.Sessions))
	}

	// Verify projects are appended
	if len(base.Projects) != 2 {
		t.Errorf("Projects count: got %d, want 2", len(base.Projects))
	}

	// Verify tool usage is merged
	if base.ToolUsage["Read"] != 18 {
		t.Errorf("ToolUsage[Read]: got %d, want 18", base.ToolUsage["Read"])
	}
	if base.ToolUsage["Edit"] != 3 {
		t.Errorf("ToolUsage[Edit]: got %d, want 3", base.ToolUsage["Edit"])
	}

	// Verify models are merged
	if base.Models["claude-opus-4-6"] != 5 {
		t.Errorf("Models[claude-opus-4-6]: got %d, want 5", base.Models["claude-opus-4-6"])
	}
	if base.Models["claude-haiku-4-5"] != 1 {
		t.Errorf("Models[claude-haiku-4-5]: got %d, want 1", base.Models["claude-haiku-4-5"])
	}

	// Verify temporals are summed
	if base.ActiveSessions != 2 {
		t.Errorf("ActiveSessions: got %d, want 2", base.ActiveSessions)
	}
	if base.TodaySessions != 2 {
		t.Errorf("TodaySessions: got %d, want 2", base.TodaySessions)
	}
}

func TestStatsMergeNil(t *testing.T) {
	base := &Stats{
		TotalSessions: 5,
	}

	// Merging nil should be a no-op
	base.Merge(nil)

	if base.TotalSessions != 5 {
		t.Errorf("Merge(nil) should not change stats, got TotalSessions=%d", base.TotalSessions)
	}
}

func TestStatsMergeDailyCosts(t *testing.T) {
	today := time.Now()
	yesterday := time.Now().AddDate(0, 0, -1)

	base := &Stats{
		DailyCosts: []DayCost{
			{Date: today, InputTokens: 1000, OutputTokens: 500, Cost: 1.50},
			{Date: yesterday, InputTokens: 800, OutputTokens: 400, Cost: 1.20},
		},
	}

	remote := &Stats{
		DailyCosts: []DayCost{
			{Date: today, InputTokens: 2000, OutputTokens: 1000, Cost: 3.00},
		},
	}

	base.Merge(remote)

	// Today should be merged
	if len(base.DailyCosts) != 2 {
		t.Fatalf("DailyCosts count: got %d, want 2", len(base.DailyCosts))
	}

	// Find today's entry
	var todayCost *DayCost
	for i := range base.DailyCosts {
		if base.DailyCosts[i].Date.Format("2006-01-02") == today.Format("2006-01-02") {
			todayCost = &base.DailyCosts[i]
			break
		}
	}

	if todayCost == nil {
		t.Fatal("Today's entry not found in DailyCosts")
	}

	if todayCost.InputTokens != 3000 {
		t.Errorf("Today InputTokens: got %d, want 3000", todayCost.InputTokens)
	}
	if todayCost.OutputTokens != 1500 {
		t.Errorf("Today OutputTokens: got %d, want 1500", todayCost.OutputTokens)
	}
	if todayCost.Cost != 4.50 {
		t.Errorf("Today Cost: got %.2f, want 4.50", todayCost.Cost)
	}
}
