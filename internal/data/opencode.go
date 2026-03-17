package data

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// OpenCode message JSON data structure.
type ocMessageData struct {
	Role   string `json:"role"`
	Model  struct {
		ProviderID string `json:"providerID"`
		ModelID    string `json:"modelID"`
	} `json:"model"`
	Tokens struct {
		Input  int `json:"input"`
		Output int `json:"output"`
		Total  int `json:"total"`
	} `json:"tokens"`
	Cost float64 `json:"cost"`
	Time struct {
		Created   int64 `json:"created"`
		Completed int64 `json:"completed"`
	} `json:"time"`
}

func openCodeDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "opencode", "opencode.db")
}

// LoadOpenCodeSessions charge les sessions depuis la DB OpenCode.
func LoadOpenCodeSessions() ([]Session, error) {
	dbPath := openCodeDBPath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, nil
	}

	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Charger les sessions
	rows, err := db.Query(`
		SELECT s.id, s.slug, s.directory, s.title, s.version, s.time_created, s.time_updated
		FROM session s
		ORDER BY s.time_created DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type sessionRow struct {
		id, slug, dir, title, version string
		created, updated             int64
	}
	var sessionRows []sessionRow

	for rows.Next() {
		var sr sessionRow
		if err := rows.Scan(&sr.id, &sr.slug, &sr.dir, &sr.title, &sr.version, &sr.created, &sr.updated); err != nil {
			continue
		}
		sessionRows = append(sessionRows, sr)
	}

	var sessions []Session
	for _, sr := range sessionRows {
		s := Session{
			ID:        sr.id,
			Source:    "opencode",
			Slug:      sr.slug,
			Project:   projectFromDir(sr.dir),
			StartTime: time.UnixMilli(sr.created),
			EndTime:   time.UnixMilli(sr.updated),
			Version:   sr.version,
			ToolUses:  make(map[string]int),
			Models:    make(map[string]int),
			PerDay:    make(map[dayKey]*DayTokens),
		}

		// Charger les messages de cette session
		msgRows, err := db.Query(`
			SELECT data, time_created FROM message WHERE session_id = ? ORDER BY time_created
		`, sr.id)
		if err != nil {
			continue
		}

		for msgRows.Next() {
			var dataStr string
			var msgTime int64
			if err := msgRows.Scan(&dataStr, &msgTime); err != nil {
				continue
			}

			var md ocMessageData
			if err := json.Unmarshal([]byte(dataStr), &md); err != nil {
				continue
			}

			model := md.Model.ModelID
			if model == "" {
				model = md.Model.ProviderID
			}

			switch md.Role {
			case "user":
				s.UserMessages++
			case "assistant":
				s.AssistantMessages++
				if model != "" {
					s.Models[model]++
				}
				s.InputTokens += md.Tokens.Input
				s.OutputTokens += md.Tokens.Output
				s.CacheReadTokens += 0 // pas dispo dans OpenCode

				msgCost := ComputeCost(model, md.Tokens.Input, md.Tokens.Output, 0, 0)
				s.Cost += msgCost

				ts := time.UnixMilli(msgTime)
				if !ts.IsZero() {
					dk := dayKeyFrom(ts)
					dt := s.PerDay[dk]
					if dt == nil {
						dt = &DayTokens{}
						s.PerDay[dk] = dt
					}
					dt.Input += md.Tokens.Input
					dt.Output += md.Tokens.Output
					dt.Messages++
					dt.Cost += msgCost
				}
			}
		}
		msgRows.Close()

		if !s.StartTime.IsZero() && !s.EndTime.IsZero() {
			s.TotalDuration = s.EndTime.Sub(s.StartTime)
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}

func projectFromDir(dir string) string {
	parts := strings.Split(dir, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return dir
}
