package data

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Codex JSONL entry structures.
type codexEntry struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexSessionMeta struct {
	ID            string `json:"id"`
	CWD           string `json:"cwd"`
	CLIVersion    string `json:"cli_version"`
	Source        string `json:"source"`
	ModelProvider string `json:"model_provider"`
	Git           struct {
		Branch string `json:"branch"`
	} `json:"git"`
}

type codexTokenUsage struct {
	InputTokens            int `json:"input_tokens"`
	CachedInputTokens      int `json:"cached_input_tokens"`
	OutputTokens           int `json:"output_tokens"`
	ReasoningOutputTokens  int `json:"reasoning_output_tokens"`
	TotalTokens            int `json:"total_tokens"`
}

type codexResponseItem struct {
	Role  string `json:"role"`
	Type  string `json:"type"`
}


func codexDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex", "state_5.sqlite")
}

func codexSessionsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex", "sessions")
}

// LoadCodexSessions charge les sessions depuis la DB Codex + JSONL.
func LoadCodexSessions() ([]Session, error) {
	dbPath := codexDBPath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, nil
	}

	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, rollout_path, model_provider, tokens_used, cwd, title,
		       created_at, updated_at, cli_version, COALESCE(git_branch, '')
		FROM threads
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var id, rolloutPath, provider, cwd, title, version, branch string
		var tokensUsed int
		var created, updated int64

		if err := rows.Scan(&id, &rolloutPath, &provider, &tokensUsed, &cwd, &title, &created, &updated, &version, &branch); err != nil {
			continue
		}

		s := Session{
			ID:        id,
			Source:    "codex",
			Slug:      truncateTitle(title, 40),
			Project:   projectFromDir(cwd),
			StartTime: time.Unix(created, 0),
			EndTime:   time.Unix(updated, 0),
			Version:   version,
			GitBranch: branch,
			ToolUses:  make(map[string]int),
			Models:    make(map[string]int),
			PerDay:    make(map[dayKey]*DayTokens),
		}

		// Enrichir depuis le JSONL si disponible
		jsonlPath := expandRolloutPath(rolloutPath)
		if jsonlPath != "" {
			enrichCodexSession(&s, jsonlPath)
		}

		// Fallback: utiliser tokens_used de la DB si pas de données JSONL
		if s.InputTokens == 0 && s.OutputTokens == 0 && tokensUsed > 0 {
			// Estimation: ~80% input, ~20% output
			s.InputTokens = tokensUsed * 80 / 100
			s.OutputTokens = tokensUsed * 20 / 100
			model := "gpt-5.3-codex" // default codex model
			s.Models[model] = 1
			s.Cost = ComputeCost(model, s.InputTokens, s.OutputTokens, 0, 0)

			dk := dayKeyFrom(s.StartTime)
			s.PerDay[dk] = &DayTokens{
				Input: s.InputTokens, Output: s.OutputTokens,
				Messages: 1, Cost: s.Cost,
			}
		}

		if !s.StartTime.IsZero() && !s.EndTime.IsZero() {
			s.TotalDuration = s.EndTime.Sub(s.StartTime)
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}

func enrichCodexSession(s *Session, path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 512*1024), 5*1024*1024)

	var model string

	for scanner.Scan() {
		var entry codexEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		switch entry.Type {
		case "session_meta":
			var meta codexSessionMeta
			if err := json.Unmarshal(entry.Payload, &meta); err == nil {
				if meta.Git.Branch != "" {
					s.GitBranch = meta.Git.Branch
				}
			}

		case "turn_context":
			// Extraire le modèle
			var tc struct {
				Model string `json:"model"`
			}
			if err := json.Unmarshal(entry.Payload, &tc); err == nil && tc.Model != "" {
				model = tc.Model
			}

		case "response_item":
			var ri codexResponseItem
			if err := json.Unmarshal(entry.Payload, &ri); err == nil {
				switch ri.Role {
				case "user":
					s.UserMessages++
				case "assistant":
					s.AssistantMessages++
				}
			}

		case "event_msg":
			var em struct {
				Type string `json:"type"`
				Info struct {
					TotalTokenUsage *codexTokenUsage `json:"total_token_usage"`
				} `json:"info"`
			}
			if err := json.Unmarshal(entry.Payload, &em); err == nil {
				if em.Type == "token_count" && em.Info.TotalTokenUsage != nil {
					tu := em.Info.TotalTokenUsage
					s.InputTokens = tu.InputTokens
					s.OutputTokens = tu.OutputTokens
					s.CacheReadTokens = tu.CachedInputTokens
					if model != "" {
						s.Models[model]++
					}
					msgCost := ComputeCost(model, tu.InputTokens, tu.OutputTokens, tu.CachedInputTokens, 0)
					s.Cost = msgCost

					ts := s.StartTime
					if !ts.IsZero() {
						dk := dayKeyFrom(ts)
						dt := s.PerDay[dk]
						if dt == nil {
							dt = &DayTokens{}
							s.PerDay[dk] = dt
						}
						dt.Input = tu.InputTokens
						dt.Output = tu.OutputTokens
						dt.CacheR = tu.CachedInputTokens
						dt.Messages++
						dt.Cost = msgCost
					}
				}
			}
		}
	}
}

func expandRolloutPath(rolloutPath string) string {
	if rolloutPath == "" {
		return ""
	}
	// Expand ~ to home dir
	if strings.HasPrefix(rolloutPath, "~") {
		home, _ := os.UserHomeDir()
		rolloutPath = filepath.Join(home, rolloutPath[1:])
	}
	if _, err := os.Stat(rolloutPath); os.IsNotExist(err) {
		return ""
	}
	return rolloutPath
}

func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen-1] + "…"
}
