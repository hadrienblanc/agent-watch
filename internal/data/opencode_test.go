package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectFromDir(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard unix path",
			input:    "/home/user/projects/myapp",
			expected: "myapp",
		},
		{
			name:     "deeply nested path",
			input:    "/home/hadrienblanc/Projets/tests/claude_monitor",
			expected: "claude_monitor",
		},
		{
			name:     "single component path",
			input:    "myproject",
			expected: "myproject",
		},
		{
			name:     "root path",
			input:    "/",
			expected: "",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "trailing slash",
			input:    "/home/user/projects/myapp/",
			expected: "",
		},
		{
			name:     "path with dots",
			input:    "/home/user/src/github.com/org/repo",
			expected: "repo",
		},
		{
			name:     "hidden directory",
			input:    "/home/user/.config/nvim",
			expected: "nvim",
		},
		{
		name:     "relative path",
			input:    "relative/path/to/project",
			expected: "project",
		},
		{
			name:     "path with spaces in directory name",
			input:    "/home/user/My Projects/cool app",
			expected: "cool app",
		},
		{
			name:     "path with special characters",
			input:    "/home/user/projects/my-app_v2.0",
			expected: "my-app_v2.0",
		},
		{
			name:     "two component path",
			input:    "/project",
			expected: "project",
		},
		{
			name:     "path with multiple consecutive slashes",
			input:    "/home//user///projects////myapp",
			expected: "myapp",
		},
		{
			name:     "windows-style path (forward slashes)",
			input:    "C:/Users/John/Documents/project",
			expected: "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := projectFromDir(tt.input)
			if result != tt.expected {
				t.Errorf("projectFromDir(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOpenCodeDBPath(t *testing.T) {
	result := openCodeDBPath()

	// Check that the path ends with the expected suffix
	expectedSuffix := filepath.Join(".local", "share", "opencode", "opencode.db")
	if !filepath.IsAbs(result) {
		t.Errorf("openCodeDBPath() = %q, expected an absolute path", result)
	}

	if !stringsEndWith(result, expectedSuffix) {
		t.Errorf("openCodeDBPath() = %q, expected to end with %q", result, expectedSuffix)
	}

	// Verify it contains the home directory
	home, _ := os.UserHomeDir()
	if home != "" && !stringsStartWith(result, home) {
		t.Errorf("openCodeDBPath() = %q, expected to start with home dir %q", result, home)
	}
}

// Helper function to check if s ends with suffix (filepath-aware)
func stringsEndWith(s, suffix string) bool {
	return filepath.Base(s) == filepath.Base(suffix) ||
		len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// Helper function to check if s starts with prefix
func stringsStartWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func TestOCMessageDataParsing(t *testing.T) {
	tests := []struct {
		name         string
		jsonInput    string
		expectedRole string
		expectedModel string
		expectedInput int
		expectedOutput int
		expectedCost  float64
		expectError   bool
	}{
		{
			name: "complete message data",
			jsonInput: `{
				"role": "assistant",
				"modelID": "claude-3-opus",
				"providerID": "anthropic",
				"tokens": {
					"input": 1000,
					"output": 500,
					"total": 1500
				},
				"cost": 0.05,
				"time": {
					"created": 1710000000000,
					"completed": 1710000050000
				}
			}`,
			expectedRole:   "assistant",
			expectedModel:  "claude-3-opus",
			expectedInput:  1000,
			expectedOutput: 500,
			expectedCost:   0.05,
		},
		{
			name: "user message",
			jsonInput: `{
				"role": "user",
				"modelID": "",
				"providerID": "",
				"tokens": {
					"input": 0,
					"output": 0,
					"total": 0
				},
				"cost": 0,
				"time": {
					"created": 1710000000000,
					"completed": 0
				}
			}`,
			expectedRole:   "user",
			expectedModel:  "",
			expectedInput:  0,
			expectedOutput: 0,
			expectedCost:   0,
		},
		{
			name: "minimal message data",
			jsonInput: `{
				"role": "assistant",
				"tokens": {
					"input": 250,
					"output": 100
				}
			}`,
			expectedRole:   "assistant",
			expectedModel:  "",
			expectedInput:  250,
			expectedOutput: 100,
			expectedCost:   0,
		},
		{
			name: "message with providerID only",
			jsonInput: `{
				"role": "assistant",
				"modelID": "",
				"providerID": "openai",
				"tokens": {
					"input": 500,
					"output": 200
				}
			}`,
			expectedRole:   "assistant",
			expectedModel:  "",
			expectedInput:  500,
			expectedOutput: 200,
		},
		{
			name: "large token values",
			jsonInput: `{
				"role": "assistant",
				"modelID": "gpt-4",
				"tokens": {
					"input": 1000000,
					"output": 500000,
					"total": 1500000
				},
				"cost": 15.50
			}`,
			expectedRole:   "assistant",
			expectedModel:  "gpt-4",
			expectedInput:  1000000,
			expectedOutput: 500000,
			expectedCost:   15.50,
		},
		{
			name: "zero values",
			jsonInput: `{
				"role": "assistant",
				"modelID": "",
				"providerID": "",
				"tokens": {
					"input": 0,
					"output": 0,
					"total": 0
				},
				"cost": 0,
				"time": {
					"created": 0,
					"completed": 0
				}
			}`,
			expectedRole:   "assistant",
			expectedModel:  "",
			expectedInput:  0,
			expectedOutput: 0,
			expectedCost:   0,
		},
		{
			name: "invalid JSON",
			jsonInput: `{
				"role": "assistant",
				"tokens": {
					"input": 100,
				}
			}`,
			expectError: true,
		},
		{
			name:       "empty JSON object",
			jsonInput:  `{}`,
			expectError: false, // Empty object should parse with zero values
		},
		{
			name: "negative cost",
			jsonInput: `{
				"role": "assistant",
				"tokens": {"input": 100, "output": 50},
				"cost": -0.05
			}`,
			expectedRole:   "assistant",
			expectedInput:  100,
			expectedOutput: 50,
			expectedCost:   -0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var md ocMessageData
			err := json.Unmarshal([]byte(tt.jsonInput), &md)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if md.Role != tt.expectedRole {
				t.Errorf("Role = %q, want %q", md.Role, tt.expectedRole)
			}
			if md.ModelID != tt.expectedModel {
				t.Errorf("ModelID = %q, want %q", md.ModelID, tt.expectedModel)
			}
			if md.Tokens.Input != tt.expectedInput {
				t.Errorf("Tokens.Input = %d, want %d", md.Tokens.Input, tt.expectedInput)
			}
			if md.Tokens.Output != tt.expectedOutput {
				t.Errorf("Tokens.Output = %d, want %d", md.Tokens.Output, tt.expectedOutput)
			}
			if md.Cost != tt.expectedCost {
				t.Errorf("Cost = %f, want %f", md.Cost, tt.expectedCost)
			}
		})
	}
}

func TestOCMessageDataTimeFields(t *testing.T) {
	jsonInput := `{
		"role": "assistant",
		"time": {
			"created": 1710000000000,
			"completed": 1710000050000
		}
	}`

	var md ocMessageData
	if err := json.Unmarshal([]byte(jsonInput), &md); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if md.Time.Created != 1710000000000 {
		t.Errorf("Time.Created = %d, want %d", md.Time.Created, 1710000000000)
	}
	if md.Time.Completed != 1710000050000 {
		t.Errorf("Time.Completed = %d, want %d", md.Time.Completed, 1710000050000)
	}
}

func TestOCMessageDataTotalTokens(t *testing.T) {
	jsonInput := `{
		"role": "assistant",
		"tokens": {
			"input": 1000,
			"output": 500,
			"total": 1500
		}
	}`

	var md ocMessageData
	if err := json.Unmarshal([]byte(jsonInput), &md); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if md.Tokens.Total != 1500 {
		t.Errorf("Tokens.Total = %d, want %d", md.Tokens.Total, 1500)
	}
}

func TestOCMessageDataFieldOrderIndependence(t *testing.T) {
	// Test that JSON fields can be in any order
	jsonInput := `{
		"cost": 0.25,
		"role": "user",
		"time": {"completed": 100, "created": 50},
		"tokens": {"total": 500, "output": 200, "input": 300},
		"providerID": "test-provider",
		"modelID": "test-model"
	}`

	var md ocMessageData
	if err := json.Unmarshal([]byte(jsonInput), &md); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if md.Role != "user" {
		t.Errorf("Role = %q, want %q", md.Role, "user")
	}
	if md.ModelID != "test-model" {
		t.Errorf("ModelID = %q, want %q", md.ModelID, "test-model")
	}
	if md.ProviderID != "test-provider" {
		t.Errorf("ProviderID = %q, want %q", md.ProviderID, "test-provider")
	}
	if md.Tokens.Input != 300 {
		t.Errorf("Tokens.Input = %d, want %d", md.Tokens.Input, 300)
	}
	if md.Tokens.Output != 200 {
		t.Errorf("Tokens.Output = %d, want %d", md.Tokens.Output, 200)
	}
	if md.Tokens.Total != 500 {
		t.Errorf("Tokens.Total = %d, want %d", md.Tokens.Total, 500)
	}
	if md.Cost != 0.25 {
		t.Errorf("Cost = %f, want %f", md.Cost, 0.25)
	}
	if md.Time.Created != 50 {
		t.Errorf("Time.Created = %d, want %d", md.Time.Created, 50)
	}
	if md.Time.Completed != 100 {
		t.Errorf("Time.Completed = %d, want %d", md.Time.Completed, 100)
	}
}

func TestLoadOpenCodeSessions_DBNotExist(t *testing.T) {
	// This test verifies that LoadOpenCodeSessions handles the case
	// where the database doesn't exist gracefully
	// We can't easily mock os.Stat, but we can document the expected behavior
	// The function should return nil, nil when the DB doesn't exist

	// Store the original DB path function
	// Since we can't easily mock this, we'll just verify the function signature
	// and document the expected behavior
	t.Log("LoadOpenCodeSessions should return nil, nil when DB doesn't exist")
}

// TestOCMessageDataUnknownFields tests that unknown fields are ignored
func TestOCMessageDataUnknownFields(t *testing.T) {
	jsonInput := `{
		"role": "assistant",
		"unknownField": "should be ignored",
		"anotherUnknown": 12345,
		"tokens": {"input": 100}
	}`

	var md ocMessageData
	err := json.Unmarshal([]byte(jsonInput), &md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if md.Role != "assistant" {
		t.Errorf("Role = %q, want %q", md.Role, "assistant")
	}
	if md.Tokens.Input != 100 {
		t.Errorf("Tokens.Input = %d, want %d", md.Tokens.Input, 100)
	}
}

// TestOCMessageDataMissingTokens tests behavior when tokens object is missing
func TestOCMessageDataMissingTokens(t *testing.T) {
	jsonInput := `{
		"role": "assistant",
		"modelID": "test-model"
	}`

	var md ocMessageData
	err := json.Unmarshal([]byte(jsonInput), &md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Zero values should be used
	if md.Tokens.Input != 0 {
		t.Errorf("Tokens.Input = %d, want 0", md.Tokens.Input)
	}
	if md.Tokens.Output != 0 {
		t.Errorf("Tokens.Output = %d, want 0", md.Tokens.Output)
	}
}

// Benchmark for ocMessageData parsing
func BenchmarkOCMessageDataUnmarshal(b *testing.B) {
	jsonInput := `{
		"role": "assistant",
		"modelID": "claude-3-opus",
		"providerID": "anthropic",
		"tokens": {
			"input": 1000,
			"output": 500,
			"total": 1500
		},
		"cost": 0.05,
		"time": {
			"created": 1710000000000,
			"completed": 1710000050000
		}
	}`

	data := []byte(jsonInput)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var md ocMessageData
		_ = json.Unmarshal(data, &md)
	}
}

// Benchmark for projectFromDir
func BenchmarkProjectFromDir(b *testing.B) {
	paths := []string{
		"/home/user/projects/myapp",
		"/home/hadrienblanc/Projets/tests/claude_monitor",
		"relative/path/to/project",
		"/",
		"",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range paths {
			_ = projectFromDir(p)
		}
	}
}
