package internal

import (
	"os"
	"path/filepath"
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
			// Skip tests that require external commands (tsc, pyright, etc.)
			// These would fail in CI without the tools installed
			t.Skip("Skipping integration test that requires external tools")

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
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestHandleTypecheckCommand(t *testing.T) {
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
			// Skip tests that require external commands
			t.Skip("Skipping integration test that requires external tools")

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
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
