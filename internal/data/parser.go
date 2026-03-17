package data

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry represents a line from a conversation JSONL file.
type Entry struct {
	Type       string    `json:"type"`
	UUID       string    `json:"uuid"`
	Timestamp  time.Time `json:"timestamp"`
	SessionID  string    `json:"sessionId"`
	Message    *Message  `json:"message"`
	DurationMs int       `json:"durationMs"`
	CWD        string    `json:"cwd"`
	Version    string    `json:"version"`
	Slug       string    `json:"slug"`
	GitBranch  string    `json:"gitBranch"`
}

type Message struct {
	Role       string    `json:"role"`
	Model      string    `json:"model"`
	Content    []Content `json:"content"`
	Usage      *Usage    `json:"usage"`
	StopReason string    `json:"stop_reason"`
}

type Content struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Name    string `json:"name"`
	IsError bool   `json:"is_error"`
}

type Usage struct {
	InputTokens              int          `json:"input_tokens"`
	OutputTokens             int          `json:"output_tokens"`
	CacheReadInputTokens     int          `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int          `json:"cache_creation_input_tokens"`
	CacheCreation            *CacheDetail `json:"cache_creation"`
}

type CacheDetail struct {
	Ephemeral5m int `json:"ephemeral_5m_input_tokens"`
	Ephemeral1h int `json:"ephemeral_1h_input_tokens"`
}

// dayKey represents a date truncated to the day.
type dayKey struct {
	Year  int
	Month time.Month
	Day   int
}

func dayKeyFrom(t time.Time) dayKey {
	return dayKey{t.Year(), t.Month(), t.Day()}
}

func (dk dayKey) Time() time.Time {
	return time.Date(dk.Year, dk.Month, dk.Day, 0, 0, 0, 0, time.Local)
}

// DayTokens aggregates tokens for a single day within a session.
type DayTokens struct {
	Input    int
	Output   int
	CacheR   int
	CacheW   int
	Messages int
	Cost     float64
}

// Available sources.
const (
	SourceClaude   = "claude"
	SourceOpenCode = "opencode"
	SourceCodex    = "codex"
	SourceGemini   = "gemini"
)

// Session summarizes a conversation.
type Session struct {
	ID        string
	Source    string
	Slug      string
	Project   string
	StartTime time.Time
	EndTime   time.Time
	Version   string
	GitBranch string

	UserMessages      int
	AssistantMessages int
	ToolUses          map[string]int
	ToolErrors        int

	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int

	Models        map[string]int
	TotalDuration time.Duration
	AvgLatencyMs  int

	Cost float64

	// Tokens per day (for daily cost calculation)
	PerDay map[dayKey]*DayTokens
}

// ProjectSummary summarizes a project.
type ProjectSummary struct {
	Name         string
	Path         string
	Sessions     int
	Messages     int
	Tokens       int
	InputTokens  int
	OutputTokens int
	CacheRead    int
	Cost         float64
}

// DayCost groups costs for a single day.
type DayCost struct {
	Date         time.Time
	InputTokens  int
	OutputTokens int
	CacheRead    int
	CacheWrite   int
	Sessions     int
	Messages     int
	Cost         float64
}

// ModelPricing contains the pricing per million tokens for a model.
type ModelPricing struct {
	InputPerM      float64
	OutputPerM     float64
	CacheReadPerM  float64
	CacheWritePerM float64
}

// Pricing per model ($/M tokens).
var modelPricing = map[string]ModelPricing{
	// Anthropic
	"claude-opus-4-6":   {InputPerM: 15.0, OutputPerM: 75.0, CacheReadPerM: 1.50, CacheWritePerM: 18.75},
	"claude-sonnet-4-6": {InputPerM: 3.0, OutputPerM: 15.0, CacheReadPerM: 0.30, CacheWritePerM: 3.75},
	"claude-sonnet-4-5": {InputPerM: 3.0, OutputPerM: 15.0, CacheReadPerM: 0.30, CacheWritePerM: 3.75},
	"claude-haiku-4-5":  {InputPerM: 0.80, OutputPerM: 4.0, CacheReadPerM: 0.08, CacheWritePerM: 1.0},
	"claude-haiku-3-5":  {InputPerM: 0.80, OutputPerM: 4.0, CacheReadPerM: 0.08, CacheWritePerM: 1.0},
	// ZhipuAI (GLM)
	"glm-5":         {InputPerM: 0.72, OutputPerM: 2.30, CacheReadPerM: 0.19, CacheWritePerM: 0.72},
	"glm-4.7":       {InputPerM: 0.50, OutputPerM: 2.00, CacheReadPerM: 0.13, CacheWritePerM: 0.50},
	"glm-4.7-flash": {InputPerM: 0.10, OutputPerM: 0.40, CacheReadPerM: 0.03, CacheWritePerM: 0.10},
	"glm-4.5":       {InputPerM: 0.60, OutputPerM: 2.20, CacheReadPerM: 0.11, CacheWritePerM: 0.60},
	// MiniMax
	"MiniMax-M2.5": {InputPerM: 0.30, OutputPerM: 1.20, CacheReadPerM: 0.03, CacheWritePerM: 0.375},
	// OpenAI
	"gpt-5.3-codex": {InputPerM: 2.00, OutputPerM: 8.00, CacheReadPerM: 0.50, CacheWritePerM: 2.00},
	"gpt-5.4":       {InputPerM: 2.00, OutputPerM: 8.00, CacheReadPerM: 0.50, CacheWritePerM: 2.00},
	"gpt-4.1":       {InputPerM: 2.00, OutputPerM: 8.00, CacheReadPerM: 0.50, CacheWritePerM: 2.00},
	"gpt-4o":        {InputPerM: 2.50, OutputPerM: 10.0, CacheReadPerM: 1.25, CacheWritePerM: 2.50},
	"gpt-4o-mini":   {InputPerM: 0.15, OutputPerM: 0.60, CacheReadPerM: 0.075, CacheWritePerM: 0.15},
	"o3":            {InputPerM: 2.00, OutputPerM: 8.00, CacheReadPerM: 0.50, CacheWritePerM: 2.00},
	"o4-mini":       {InputPerM: 1.10, OutputPerM: 4.40, CacheReadPerM: 0.275, CacheWritePerM: 1.10},
	// Google
	"gemini-2.5-pro":         {InputPerM: 1.25, OutputPerM: 10.0, CacheReadPerM: 0.315, CacheWritePerM: 4.50},
	"gemini-2.5-flash":       {InputPerM: 0.15, OutputPerM: 0.60, CacheReadPerM: 0.0375, CacheWritePerM: 0.15},
	"gemini-2.5-flash-lite":  {InputPerM: 0.05, OutputPerM: 0.20, CacheReadPerM: 0.0125, CacheWritePerM: 0.05},
	"gemini-3-flash-preview": {InputPerM: 0.15, OutputPerM: 0.60, CacheReadPerM: 0.0375, CacheWritePerM: 0.15},
	"gemini-2.0-flash":       {InputPerM: 0.10, OutputPerM: 0.40, CacheReadPerM: 0.025, CacheWritePerM: 0.10},
	// Zero cost (synthetic)
	"<synthetic>": {InputPerM: 0, OutputPerM: 0, CacheReadPerM: 0, CacheWritePerM: 0},
}

// Default pricing (Opus) for unknown models.
var defaultPricing = modelPricing["claude-opus-4-6"]

func pricingFor(model string) ModelPricing {
	// Look for an exact or prefix match
	if p, ok := modelPricing[model]; ok {
		return p
	}
	for prefix, p := range modelPricing {
		if strings.HasPrefix(model, prefix) {
			return p
		}
	}
	return defaultPricing
}

// ComputeCost calculates the estimated cost in dollars for a given model.
// inputTokens = non-cached tokens (cache miss), cacheRead = tokens read from cache,
// cacheWrite = tokens written to cache (cache_creation_input_tokens).
func ComputeCost(model string, inputTokens, outputTokens, cacheRead, cacheWrite int) float64 {
	p := pricingFor(model)
	return float64(inputTokens)*p.InputPerM/1_000_000 +
		float64(outputTokens)*p.OutputPerM/1_000_000 +
		float64(cacheRead)*p.CacheReadPerM/1_000_000 +
		float64(cacheWrite)*p.CacheWritePerM/1_000_000
}

// Stats aggregates all data for the dashboard.
type Stats struct {
	Sessions  []Session
	Projects  []ProjectSummary
	ToolUsage map[string]int

	TotalInputTokens  int
	TotalOutputTokens int
	TotalCacheRead    int
	TotalMessages     int
	TotalToolUses     int
	TotalToolErrors   int
	TotalSessions     int

	// Temporal
	ActiveSessions int // activity in the last 30 minutes
	TodaySessions  int
	TodayMessages  int
	TodayTokens    int
	WeekSessions   int
	WeekMessages   int
	WeekTokens     int

	// Daily costs (last 60 days)
	DailyCosts []DayCost
	TotalCost  float64

	ActiveModel string
	Models      map[string]int

	LastUpdated time.Time
}

func claudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// aggregateSession aggregates session metrics into global stats.
func aggregateSession(stats *Stats, s *Session) {
	stats.TotalInputTokens += s.InputTokens
	stats.TotalOutputTokens += s.OutputTokens
	stats.TotalCacheRead += s.CacheReadTokens
	stats.TotalMessages += s.UserMessages + s.AssistantMessages
	stats.TotalToolErrors += s.ToolErrors

	for tool, count := range s.ToolUses {
		stats.ToolUsage[tool] += count
		stats.TotalToolUses += count
	}
	for model, count := range s.Models {
		stats.Models[model] += count
	}
}

// aggregateProject aggregates session metrics into a project summary.
func aggregateProject(ps *ProjectSummary, s *Session) {
	ps.Sessions++
	ps.Messages += s.UserMessages + s.AssistantMessages
	ps.Tokens += s.InputTokens + s.OutputTokens
	ps.InputTokens += s.InputTokens
	ps.OutputTokens += s.OutputTokens
	ps.CacheRead += s.CacheReadTokens
	ps.Cost += s.Cost
}

// LoadStats loads and aggregates all conversations.
// NewStats creates an empty Stats ready for incremental loading.
func NewStats() *Stats {
	return &Stats{
		ToolUsage: make(map[string]int),
		Models:    make(map[string]int),
	}
}

// LoadClaudeSessions loads Claude Code conversations into stats.
func LoadClaudeSessions(stats *Stats) {
	projectsDir := filepath.Join(claudeDir(), "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectPath := filepath.Join(projectsDir, entry.Name())
		projectName := decodeProjectName(entry.Name())

		sessions, err := loadProjectSessions(projectPath)
		if err != nil {
			continue
		}

		projSummary := ProjectSummary{
			Name: projectName,
			Path: projectPath,
		}

		for i := range sessions {
			sessions[i].Project = projectName
			sessions[i].Source = SourceClaude
			s := &sessions[i]
			aggregateSession(stats, s)
			aggregateProject(&projSummary, s)
		}

		stats.Sessions = append(stats.Sessions, sessions...)
		stats.Projects = append(stats.Projects, projSummary)
	}
}

// LoadExternalSource loads sessions from an external source (opencode, codex, gemini) into stats.
func LoadExternalSource(stats *Stats, name string, loader func() ([]Session, error)) {
	extraSessions, err := loader()
	if err != nil || len(extraSessions) == 0 {
		return
	}

	projMap := make(map[string]*ProjectSummary)
	for i := range extraSessions {
		s := &extraSessions[i]
		aggregateSession(stats, s)

		key := name + "/" + s.Project
		ps := projMap[key]
		if ps == nil {
			ps = &ProjectSummary{Name: s.Project + " (" + name + ")"}
			projMap[key] = ps
		}
		aggregateProject(ps, s)
	}

	stats.Sessions = append(stats.Sessions, extraSessions...)
	for _, ps := range projMap {
		stats.Projects = append(stats.Projects, *ps)
	}
}

// LoadStats loads and aggregates all conversations.
func LoadStats() (*Stats, error) {
	stats := NewStats()
	LoadClaudeSessions(stats)
	LoadExternalSource(stats, "opencode", LoadOpenCodeSessions)
	LoadExternalSource(stats, "codex", LoadCodexSessions)
	LoadExternalSource(stats, "gemini", LoadGeminiSessions)
	FinalizeStats(stats)
	return stats, nil
}

// FinalizeStats computes derived fields (counters, sorts, daily costs) after all sessions are loaded.
func FinalizeStats(stats *Stats) {
	stats.TotalSessions = len(stats.Sessions)

	// Temporal counters
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(todayStart.Weekday()-time.Monday))
	if todayStart.Weekday() == time.Sunday {
		weekStart = todayStart.AddDate(0, 0, -6)
	}
	activeThreshold := now.Add(-30 * time.Minute)

	for _, s := range stats.Sessions {
		msgs := s.UserMessages + s.AssistantMessages
		tokens := s.InputTokens + s.OutputTokens

		if s.EndTime.After(activeThreshold) {
			stats.ActiveSessions++
		}
		if s.StartTime.After(todayStart) || s.EndTime.After(todayStart) {
			stats.TodaySessions++
			stats.TodayMessages += msgs
			stats.TodayTokens += tokens
		}
		if s.StartTime.After(weekStart) || s.EndTime.After(weekStart) {
			stats.WeekSessions++
			stats.WeekMessages += msgs
			stats.WeekTokens += tokens
		}
	}

	// Sort sessions by descending date
	sort.Slice(stats.Sessions, func(i, j int) bool {
		return stats.Sessions[i].StartTime.After(stats.Sessions[j].StartTime)
	})

	// Sort projects by descending messages
	sort.Slice(stats.Projects, func(i, j int) bool {
		return stats.Projects[i].Messages > stats.Projects[j].Messages
	})

	// Primary model
	maxCount := 0
	for model, count := range stats.Models {
		if count > maxCount {
			maxCount = count
			stats.ActiveModel = model
		}
	}

	// Daily costs (last 60 days)
	dayMap := make(map[dayKey]*DayCost)
	cutoff := now.AddDate(0, 0, -60)

	for _, sess := range stats.Sessions {
		for dk, dt := range sess.PerDay {
			if dk.Time().Before(cutoff) {
				continue
			}
			dc := dayMap[dk]
			if dc == nil {
				dc = &DayCost{Date: dk.Time()}
				dayMap[dk] = dc
			}
			dc.InputTokens += dt.Input
			dc.OutputTokens += dt.Output
			dc.CacheRead += dt.CacheR
			dc.CacheWrite += dt.CacheW
			dc.Messages += dt.Messages
			dc.Cost += dt.Cost
			dc.Sessions++
		}
	}

	// Fill days without activity and sort
	for d := cutoff; !d.After(now); d = d.AddDate(0, 0, 1) {
		dk := dayKeyFrom(d)
		if _, ok := dayMap[dk]; !ok {
			dayMap[dk] = &DayCost{Date: dk.Time()}
		}
	}

	stats.DailyCosts = make([]DayCost, 0, len(dayMap))
	for _, dc := range dayMap {
		stats.DailyCosts = append(stats.DailyCosts, *dc)
	}
	sort.Slice(stats.DailyCosts, func(i, j int) bool {
		return stats.DailyCosts[i].Date.Before(stats.DailyCosts[j].Date)
	})

	for _, sess := range stats.Sessions {
		stats.TotalCost += sess.Cost
	}

	stats.LastUpdated = time.Now()
}

func loadProjectSessions(projectDir string) ([]Session, error) {
	files, err := filepath.Glob(filepath.Join(projectDir, "*.jsonl"))
	if err != nil {
		return nil, err
	}

	var sessions []Session
	for _, f := range files {
		// Skip memory file
		if strings.Contains(filepath.Base(f), "memory") {
			continue
		}
		s, err := parseSession(f)
		if err != nil {
			continue
		}
		sessions = append(sessions, *s)
	}
	return sessions, nil
}

func parseSession(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := &Session{
		ID:       strings.TrimSuffix(filepath.Base(path), ".jsonl"),
		ToolUses: make(map[string]int),
		Models:   make(map[string]int),
		PerDay:   make(map[dayKey]*DayTokens),
	}

	var latencies []int
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB buffer

	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		// Timestamps
		if !entry.Timestamp.IsZero() {
			if s.StartTime.IsZero() || entry.Timestamp.Before(s.StartTime) {
				s.StartTime = entry.Timestamp
			}
			if entry.Timestamp.After(s.EndTime) {
				s.EndTime = entry.Timestamp
			}
		}

		if entry.Slug != "" && s.Slug == "" {
			s.Slug = entry.Slug
		}
		if entry.SessionID != "" {
			s.ID = entry.SessionID
		}
		if entry.Version != "" {
			s.Version = entry.Version
		}
		if entry.GitBranch != "" {
			s.GitBranch = entry.GitBranch
		}

		if entry.DurationMs > 0 {
			latencies = append(latencies, entry.DurationMs)
		}

		if entry.Message == nil {
			continue
		}

		switch entry.Message.Role {
		case "user":
			s.UserMessages++
		case "assistant":
			s.AssistantMessages++
			if entry.Message.Model != "" {
				s.Models[entry.Message.Model]++
			}
			if entry.Message.Usage != nil {
				u := entry.Message.Usage
				s.InputTokens += u.InputTokens
				s.OutputTokens += u.OutputTokens
				s.CacheReadTokens += u.CacheReadInputTokens
				s.CacheCreationTokens += u.CacheCreationInputTokens

				msgCost := ComputeCost(entry.Message.Model, u.InputTokens, u.OutputTokens, u.CacheReadInputTokens, u.CacheCreationInputTokens)
				s.Cost += msgCost

				// Aggregate by day
				if !entry.Timestamp.IsZero() {
					dk := dayKeyFrom(entry.Timestamp)
					dt := s.PerDay[dk]
					if dt == nil {
						dt = &DayTokens{}
						s.PerDay[dk] = dt
					}
					dt.Input += u.InputTokens
					dt.Output += u.OutputTokens
					dt.CacheR += u.CacheReadInputTokens
					dt.CacheW += u.CacheCreationInputTokens
					dt.Messages++
					dt.Cost += msgCost
				}
			}

			for _, c := range entry.Message.Content {
				if c.Type == "tool_use" && c.Name != "" {
					s.ToolUses[c.Name]++
				}
				if c.Type == "tool_result" && c.IsError {
					s.ToolErrors++
				}
			}
		}
	}

	if !s.StartTime.IsZero() && !s.EndTime.IsZero() {
		s.TotalDuration = s.EndTime.Sub(s.StartTime)
	}

	if len(latencies) > 0 {
		total := 0
		for _, l := range latencies {
			total += l
		}
		s.AvgLatencyMs = total / len(latencies)
	}

	return s, nil
}

func decodeProjectName(encoded string) string {
	// "-home-hadrienblanc-Projets-tests-form-on-terminal" -> "form-on-terminal"
	parts := strings.Split(encoded, "-")
	// Take the last 2-3 significant segments
	if len(parts) > 2 {
		// Find the last significant non-empty segment
		result := parts[len(parts)-1]
		for i := len(parts) - 2; i >= 0; i-- {
			if parts[i] == "" {
				continue
			}
			// Stop at words like "Projets", "home", etc.
			lower := strings.ToLower(parts[i])
			if lower == "projets" || lower == "home" || lower == "tests" || lower == "" {
				break
			}
			result = parts[i] + "-" + result
		}
		if result != "" {
			return result
		}
	}
	return encoded
}
