package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Gemini session JSON structures.
type geminiSession struct {
	SessionID   string          `json:"sessionId"`
	StartTime   string          `json:"startTime"`
	LastUpdated string          `json:"lastUpdated"`
	Messages    []geminiMessage `json:"messages"`
}

type geminiMessage struct {
	Type      string        `json:"type"`
	Timestamp string        `json:"timestamp"`
	Model     string        `json:"model"`
	Tokens    *geminiTokens `json:"tokens"`
	ToolCalls []struct {
		Name string `json:"name"`
	} `json:"toolCalls"`
}

type geminiTokens struct {
	Input    int `json:"input"`
	Output   int `json:"output"`
	Cached   int `json:"cached"`
	Thoughts int `json:"thoughts"`
	Tool     int `json:"tool"`
	Total    int `json:"total"`
}

type geminiProjects struct {
	Projects map[string]string `json:"projects"`
}

func geminiDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gemini")
}

// LoadGeminiSessions charge les sessions depuis les fichiers JSON de Gemini CLI.
func LoadGeminiSessions() ([]Session, error) {
	baseDir := geminiDir()
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Charger le mapping projets
	projMap := loadGeminiProjects(filepath.Join(baseDir, "projects.json"))

	// Inverser: slug -> project name (dernier segment du path)
	slugToProject := make(map[string]string)
	for path, slug := range projMap {
		slugToProject[slug] = projectFromDir(path)
	}

	// Scanner tous les dossiers dans tmp/
	tmpDir := filepath.Join(baseDir, "tmp")
	slugDirs, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, nil
	}

	var sessions []Session
	for _, slugDir := range slugDirs {
		if !slugDir.IsDir() {
			continue
		}
		slug := slugDir.Name()
		chatsDir := filepath.Join(tmpDir, slug, "chats")

		files, err := filepath.Glob(filepath.Join(chatsDir, "session-*.json"))
		if err != nil || len(files) == 0 {
			continue
		}

		projectName := slugToProject[slug]
		if projectName == "" {
			projectName = slug
		}

		for _, f := range files {
			s, err := parseGeminiSession(f, projectName)
			if err != nil {
				continue
			}
			sessions = append(sessions, *s)
		}
	}

	return sessions, nil
}

func loadGeminiProjects(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var gp geminiProjects
	if err := json.Unmarshal(data, &gp); err != nil {
		return nil
	}
	return gp.Projects
}

func parseGeminiSession(path, project string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var gs geminiSession
	if err := json.Unmarshal(data, &gs); err != nil {
		return nil, err
	}

	startTime, _ := time.Parse(time.RFC3339Nano, gs.StartTime)
	endTime, _ := time.Parse(time.RFC3339Nano, gs.LastUpdated)

	s := &Session{
		ID:        gs.SessionID,
		Source:    "gemini",
		Slug:      filepath.Base(path),
		Project:   project,
		StartTime: startTime,
		EndTime:   endTime,
		ToolUses:  make(map[string]int),
		Models:    make(map[string]int),
		PerDay:    make(map[dayKey]*DayTokens),
	}

	for _, msg := range gs.Messages {
		switch msg.Type {
		case "user":
			s.UserMessages++
		case "gemini":
			s.AssistantMessages++

			model := msg.Model
			if model != "" {
				s.Models[model]++
			}

			if msg.Tokens != nil {
				t := msg.Tokens
				s.InputTokens += t.Input
				s.OutputTokens += t.Output
				s.CacheReadTokens += t.Cached

				msgCost := ComputeCost(model, t.Input, t.Output, t.Cached, 0)
				s.Cost += msgCost

				ts, _ := time.Parse(time.RFC3339Nano, msg.Timestamp)
				if !ts.IsZero() {
					dk := dayKeyFrom(ts)
					dt := s.PerDay[dk]
					if dt == nil {
						dt = &DayTokens{}
						s.PerDay[dk] = dt
					}
					dt.Input += t.Input
					dt.Output += t.Output
					dt.CacheR += t.Cached
					dt.Messages++
					dt.Cost += msgCost
				}
			}

			for _, tc := range msg.ToolCalls {
				if tc.Name != "" {
					s.ToolUses[tc.Name]++
				}
			}
		}
	}

	if !s.StartTime.IsZero() && !s.EndTime.IsZero() {
		s.TotalDuration = s.EndTime.Sub(s.StartTime)
	}

	return s, nil
}
