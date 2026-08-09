package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasTypecheckCapability(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(dir string)
		expected  bool
	}{
		{
			name: "TypeScript project with tsconfig.json",
			setupFunc: func(dir string) {
				os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0644)
			},
			expected: true,
		},
		{
			name: "Python project with pyright in pyproject.toml",
			setupFunc: func(dir string) {
				content := `[tool.pyright]
basic = true`
				os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)
			},
			expected: true,
		},
		{
			name: "Python project with mypy in pyproject.toml",
			setupFunc: func(dir string) {
				content := `[tool.mypy]
strict = true`
				os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)
			},
			expected: true,
		},
		{
			name: "Python project without type checker",
			setupFunc: func(dir string) {
				content := `[project]
name = "test"`
				os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)
			},
			expected: false,
		},
		{
			name: "Rust project with Cargo.toml",
			setupFunc: func(dir string) {
				os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"test\""), 0644)
			},
			expected: true,
		},
		{
			name: "Go project with go.mod",
			setupFunc: func(dir string) {
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
			},
			expected: true,
		},
		{
			name: "Project without type checking",
			setupFunc: func(dir string) {
				os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tt.setupFunc(tempDir)

			runner := &CommandRunner{
				CurrentDir:  tempDir,
				ProjectRoot: tempDir,
			}

			result := runner.hasTypecheckCapability()
			if result != tt.expected {
				t.Errorf("hasTypecheckCapability() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSynthesizeTypecheckCommand(t *testing.T) {
	installTypecheckTestCommands(t)

	tests := []struct {
		name          string
		setupFunc     func(dir string)
		expectError   bool
		errorContains string
	}{
		{
			name: "TypeScript project with tsconfig.json",
			setupFunc: func(dir string) {
				os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0644)
				os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
				os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}"), 0644)
			},
			expectError: false,
		},
		{
			name: "Python project with pyright",
			setupFunc: func(dir string) {
				content := `[project]
name = "test"

[tool.pyright]
strict = ["src"]`
				os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)
			},
			expectError: false,
		},
		{
			name: "Python project with mypy",
			setupFunc: func(dir string) {
				content := `[project]
name = "test"

[tool.mypy]
strict = true`
				os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)
			},
			expectError: false,
		},
		{
			name: "Rust project",
			setupFunc: func(dir string) {
				os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"test\""), 0644)
			},
			expectError: false,
		},
		{
			name: "Go project",
			setupFunc: func(dir string) {
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
			},
			expectError: false,
		},
		{
			name: "Project without typecheck support",
			setupFunc: func(dir string) {
				os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)
			},
			expectError:   true,
			errorContains: "could not synthesize typecheck command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tt.setupFunc(tempDir)

			runner := &CommandRunner{
				Command:     "typecheck",
				Args:        []string{},
				CurrentDir:  tempDir,
				ProjectRoot: tempDir,
			}

			err := runner.synthesizeTypecheckCommand()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestHandleTypecheckCommand(t *testing.T) {
	installTypecheckTestCommands(t)

	tests := []struct {
		name          string
		setupFunc     func(dir string)
		expectError   bool
		errorContains string
	}{
		{
			name: "Project with uv and pyright",
			setupFunc: func(dir string) {
				content := `[project]
name = "test"

[tool.uv]

[tool.pyright]
strict = ["src"]`
				os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)
				os.WriteFile(filepath.Join(dir, "uv.lock"), []byte(""), 0644)
			},
			expectError: false,
		},
		{
			name: "Project with Poetry and pyright",
			setupFunc: func(dir string) {
				content := `[tool.poetry]
name = "test"

[tool.pyright]
strict = ["src"]`
				os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)
				os.WriteFile(filepath.Join(dir, "poetry.lock"), []byte(""), 0644)
			},
			expectError: false,
		},
		{
			name: "Plain Python project with pyright (no uv/poetry)",
			setupFunc: func(dir string) {
				content := `[project]
name = "test"

[tool.pyright]
strict = ["src"]`
				os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)
			},
			expectError: false,
		},
		{
			name: "TypeScript project without package.json",
			setupFunc: func(dir string) {
				os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0644)
			},
			expectError: false,
		},
		{
			name: "Project without typecheck capability",
			setupFunc: func(dir string) {
				os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)
			},
			expectError:   true,
			errorContains: "no typecheck command or type checking capability found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tt.setupFunc(tempDir)

			runner := &CommandRunner{
				Command:     "typecheck",
				Args:        []string{},
				CurrentDir:  tempDir,
				ProjectRoot: tempDir,
			}

			err := HandleTypecheckCommand(runner)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func installTypecheckTestCommands(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"cargo", "go", "mypy", "npx", "poetry", "pyright", "uv"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
