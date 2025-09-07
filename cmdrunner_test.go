package cmdrunner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"format", "format"},
		{"fmt", "format"},
		{"run", "run"},
		{"dev", "dev"},
		{"serve", "serve"},
		{"start", "start"},
		{"build", "build"},
		{"lint", "lint"},
		{"check", "check"},
		{"test", "test"},
		{"fix", "fix"},
		{"clean", "clean"},
		{"install", "install"},
		{"setup", "setup"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeCommand(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeCommand(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetCommandVariants(t *testing.T) {
	tests := []struct {
		command  string
		expected []string
	}{
		{"format", []string{"format", "fmt"}},
		{"run", []string{"run", "dev", "serve", "start"}},
		{"lint", []string{"lint"}},
		{"check", []string{"check"}},
		{"typecheck", []string{"typecheck", "type-check", "types"}},
		{"unknown", []string{"unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := GetCommandVariants(tt.command)
			if !slicesEqual(result, tt.expected) {
				t.Errorf("getCommandVariants(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestFindProjectRoot(t *testing.T) {
	tempDir := t.TempDir()

	projectDir := filepath.Join(tempDir, "project")
	subDir := filepath.Join(projectDir, "src", "components")

	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	gitDir := filepath.Join(projectDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	runner := &CommandRunner{}

	tests := []struct {
		name     string
		startDir string
		expected string
	}{
		{"from project root", projectDir, projectDir},
		{"from subdirectory", subDir, projectDir},
		{"no vcs", tempDir, tempDir},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.FindProjectRoot(tt.startDir)
			if result != tt.expected {
				t.Errorf("findProjectRoot(%q) = %q, want %q", tt.startDir, result, tt.expected)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "exists.txt")

	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{existingFile, true},
		{filepath.Join(tempDir, "nonexistent.txt"), false},
		{tempDir, true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := FileExists(tt.path)
			if result != tt.expected {
				t.Errorf("fileExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
