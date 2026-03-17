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

// Entry est une ligne du fichier JSONL de conversation.
type Entry struct {
	Type      string    `json:"type"`
	UUID      string    `json:"uuid"`
	Timestamp time.Time `json:"timestamp"`
	SessionID string    `json:"sessionId"`
	Message   *Message  `json:"message"`
	DurationMs int      `json:"durationMs"`
	CWD       string    `json:"cwd"`
	Version   string    `json:"version"`
	Slug      string    `json:"slug"`
	GitBranch string    `json:"gitBranch"`
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

// dayKey retourne la date tronquée au jour.
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

// DayTokens agrège les tokens d'une journée dans une session.
type DayTokens struct {
	Input    int
	Output   int
	CacheR   int
	CacheW   int
	Messages int
	Cost     float64
}

// Session résume une conversation.
type Session struct {
	ID        string
	Source    string // "claude", "opencode", "codex"
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

	InputTokens              int
	OutputTokens             int
	CacheReadTokens          int
	CacheCreationTokens      int

	Models        map[string]int
	TotalDuration time.Duration
	AvgLatencyMs  int

	Cost float64

	// Tokens par jour (pour calcul coût journalier)
	PerDay map[dayKey]*DayTokens
}

// ProjectSummary résume un projet.
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

// DayCost regroupe les coûts d'une journée.
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

// ModelPricing contient le pricing par million de tokens pour un modèle.
type ModelPricing struct {
	InputPerM      float64
	OutputPerM     float64
	CacheReadPerM  float64
	CacheWritePerM float64
}

// Pricing par modèle ($/M tokens).
var modelPricing = map[string]ModelPricing{
	// Anthropic
	"claude-opus-4-6":   {InputPerM: 15.0, OutputPerM: 75.0, CacheReadPerM: 1.50, CacheWritePerM: 18.75},
	"claude-sonnet-4-6": {InputPerM: 3.0, OutputPerM: 15.0, CacheReadPerM: 0.30, CacheWritePerM: 3.75},
	"claude-haiku-4-5":  {InputPerM: 0.80, OutputPerM: 4.0, CacheReadPerM: 0.08, CacheWritePerM: 1.0},
	// ZhipuAI (GLM)
	"glm-5":         {InputPerM: 0.72, OutputPerM: 2.30, CacheReadPerM: 0.19, CacheWritePerM: 0.72},
	"glm-4.7":       {InputPerM: 0.50, OutputPerM: 2.00, CacheReadPerM: 0.13, CacheWritePerM: 0.50},
	"glm-4.7-flash": {InputPerM: 0.10, OutputPerM: 0.40, CacheReadPerM: 0.03, CacheWritePerM: 0.10},
	"glm-4.5":       {InputPerM: 0.60, OutputPerM: 2.20, CacheReadPerM: 0.11, CacheWritePerM: 0.60},
	// MiniMax
	"MiniMax-M2.5": {InputPerM: 0.30, OutputPerM: 1.20, CacheReadPerM: 0.03, CacheWritePerM: 0.375},
	// OpenAI (Codex)
	"gpt-5.3-codex": {InputPerM: 2.00, OutputPerM: 8.00, CacheReadPerM: 0.50, CacheWritePerM: 2.00},
	"gpt-5.4":       {InputPerM: 2.00, OutputPerM: 8.00, CacheReadPerM: 0.50, CacheWritePerM: 2.00},
	"gpt-4o":        {InputPerM: 2.50, OutputPerM: 10.0, CacheReadPerM: 1.25, CacheWritePerM: 2.50},
}

// Pricing par défaut (Opus) pour modèles inconnus.
var defaultPricing = modelPricing["claude-opus-4-6"]

func pricingFor(model string) ModelPricing {
	// Chercher une correspondance exacte ou par préfixe
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

// ComputeCost calcule le coût estimé en dollars pour un modèle donné.
func ComputeCost(model string, inputTokens, outputTokens, cacheRead, cacheWrite int) float64 {
	p := pricingFor(model)
	cacheMiss := inputTokens - cacheRead
	if cacheMiss < 0 {
		cacheMiss = 0
	}
	return float64(cacheMiss)*p.InputPerM/1_000_000 +
		float64(outputTokens)*p.OutputPerM/1_000_000 +
		float64(cacheRead)*p.CacheReadPerM/1_000_000 +
		float64(cacheWrite)*p.CacheWritePerM/1_000_000
}

// Stats agrège toutes les données pour le dashboard.
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

	// Temporels
	ActiveSessions  int // activité dans les 30 dernières minutes
	TodaySessions   int
	TodayMessages   int
	TodayTokens     int
	WeekSessions    int
	WeekMessages    int
	WeekTokens      int

	// Coûts par jour (60 derniers jours)
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

// LoadStats charge et agrège toutes les conversations.
func LoadStats() (*Stats, error) {
	projectsDir := filepath.Join(claudeDir(), "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	stats := &Stats{
		ToolUsage: make(map[string]int),
		Models:    make(map[string]int),
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
			Name:     projectName,
			Path:     projectPath,
			Sessions: len(sessions),
		}

		for i := range sessions {
			sessions[i].Project = projectName
			sessions[i].Source = "claude"
			s := &sessions[i]

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

			projSummary.Messages += s.UserMessages + s.AssistantMessages
			projSummary.Tokens += s.InputTokens + s.OutputTokens
			projSummary.InputTokens += s.InputTokens
			projSummary.OutputTokens += s.OutputTokens
			projSummary.CacheRead += s.CacheReadTokens
			projSummary.Cost += s.Cost
		}

		stats.Sessions = append(stats.Sessions, sessions...)
		stats.Projects = append(stats.Projects, projSummary)
	}

	// Charger OpenCode et Codex
	for _, loader := range []struct {
		name string
		fn   func() ([]Session, error)
	}{
		{"opencode", LoadOpenCodeSessions},
		{"codex", LoadCodexSessions},
	} {
		extraSessions, err := loader.fn()
		if err != nil || len(extraSessions) == 0 {
			continue
		}

		// Agréger par projet
		projMap := make(map[string]*ProjectSummary)
		for i := range extraSessions {
			s := &extraSessions[i]
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

			key := loader.name + "/" + s.Project
			ps := projMap[key]
			if ps == nil {
				ps = &ProjectSummary{Name: s.Project + " (" + loader.name + ")"}
				projMap[key] = ps
			}
			ps.Sessions++
			ps.Messages += s.UserMessages + s.AssistantMessages
			ps.Tokens += s.InputTokens + s.OutputTokens
			ps.InputTokens += s.InputTokens
			ps.OutputTokens += s.OutputTokens
			ps.CacheRead += s.CacheReadTokens
			ps.Cost += s.Cost
		}

		stats.Sessions = append(stats.Sessions, extraSessions...)
		for _, ps := range projMap {
			stats.Projects = append(stats.Projects, *ps)
		}
	}

	stats.TotalSessions = len(stats.Sessions)

	// Compteurs temporels
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

	// Trier sessions par date décroissante
	sort.Slice(stats.Sessions, func(i, j int) bool {
		return stats.Sessions[i].StartTime.After(stats.Sessions[j].StartTime)
	})

	// Trier projets par messages décroissants
	sort.Slice(stats.Projects, func(i, j int) bool {
		return stats.Projects[i].Messages > stats.Projects[j].Messages
	})

	// Modèle principal
	maxCount := 0
	for model, count := range stats.Models {
		if count > maxCount {
			maxCount = count
			stats.ActiveModel = model
		}
	}

	// Coûts par jour (60 derniers jours)
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

	// Remplir les jours sans activité et trier
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
	return stats, nil
}

func loadProjectSessions(projectDir string) ([]Session, error) {
	files, err := filepath.Glob(filepath.Join(projectDir, "*.jsonl"))
	if err != nil {
		return nil, err
	}

	var sessions []Session
	for _, f := range files {
		// Ignorer le fichier memory
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

				// Agréger par jour
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
	// Prendre les 2-3 derniers segments significatifs
	if len(parts) > 2 {
		// Trouver le dernier segment non-vide significatif
		result := parts[len(parts)-1]
		for i := len(parts) - 2; i >= 0; i-- {
			if parts[i] == "" {
				continue
			}
			// S'arrêter à des mots comme "Projets", "home", etc.
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
