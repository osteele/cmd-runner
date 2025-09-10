package cmdrunner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PoetrySource for Poetry projects
type PoetrySource struct {
	baseSource
}

func NewPoetrySource(dir string) CommandSource {
	// Verify it's actually a Poetry project
	if !FileExists(filepath.Join(dir, "poetry.lock")) {
		// Check if pyproject.toml contains [tool.poetry]
		if FileExists(filepath.Join(dir, "pyproject.toml")) {
			data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
			if err != nil || !strings.Contains(string(data), "[tool.poetry]") {
				return nil
			}
		} else {
			return nil
		}
	}
	
	return &PoetrySource{
		baseSource: baseSource{
			dir:      dir,
			name:     "Poetry",
			priority: 10,
		},
	}
}

func (p *PoetrySource) ListCommands() map[string]string {
	return map[string]string{
		"install":   "poetry install",
		"run":       "poetry run python",
		"test":      "poetry run pytest",
		"format":    "poetry run ruff format",
		"lint":      "poetry run ruff check",
		"typecheck": "poetry run pyright",
		"build":     "poetry build",
		"publish":   "poetry publish",
	}
}

func (p *PoetrySource) FindCommand(command string, args []string) *exec.Cmd {
	poetryCommands := map[string][]string{
		"install":   {"install"},
		"setup":     {"install"},
		"run":       {"run", "python"},
		"test":      {"run", "pytest"},
		"lint":      {"run", "ruff", "check"},
		"format":    {"run", "ruff", "format"},
		"fmt":       {"run", "ruff", "format"},
		"fix":       {"run", "ruff", "check", "--fix"},
		"typecheck": {"run", "pyright"},
		"tc":        {"run", "pyright"},
		"build":     {"build"},
		"publish":   {"publish"},
	}
	
	for _, variant := range GetCommandVariants(command) {
		if poetryCmd, ok := poetryCommands[variant]; ok {
			cmdArgs := append(poetryCmd, args...)
			cmd := exec.Command("poetry", cmdArgs...)
			cmd.Dir = p.dir
			return cmd
		}
	}
	
	// Try to run any command through poetry run
	cmdArgs := append([]string{"run", command}, args...)
	cmd := exec.Command("poetry", cmdArgs...)
	cmd.Dir = p.dir
	return cmd
}

// UvSource for uv projects
type UvSource struct {
	baseSource
}

func NewUvSource(dir string) CommandSource {
	// Verify it's actually a uv project
	hasUv := false
	
	if FileExists(filepath.Join(dir, "uv.lock")) || FileExists(filepath.Join(dir, ".uv")) {
		hasUv = true
	} else if FileExists(filepath.Join(dir, "pyproject.toml")) {
		data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
		if err == nil && strings.Contains(string(data), "[tool.uv]") {
			hasUv = true
		}
	}
	
	if !hasUv {
		return nil
	}
	
	return &UvSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "uv",
			priority: 10,
		},
	}
}

func (u *UvSource) ListCommands() map[string]string {
	return map[string]string{
		"install":   "uv sync",
		"run":       "uv run",
		"test":      "uv run pytest",
		"format":    "uv run ruff format",
		"lint":      "uv run ruff check",
		"typecheck": "uv run pyright",
	}
}

func (u *UvSource) FindCommand(command string, args []string) *exec.Cmd {
	uvCommands := map[string][]string{
		"install":   {"sync"},
		"setup":     {"sync"},
		"run":       {"run"},
		"test":      {"run", "pytest"},
		"lint":      {"run", "ruff", "check"},
		"format":    {"run", "ruff", "format"},
		"fmt":       {"run", "ruff", "format"},
		"fix":       {"run", "ruff", "check", "--fix"},
		"typecheck": {"run", "pyright"},
		"tc":        {"run", "pyright"},
	}
	
	for _, variant := range GetCommandVariants(command) {
		if uvCmd, ok := uvCommands[variant]; ok {
			cmdArgs := append(uvCmd, args...)
			cmd := exec.Command("uv", cmdArgs...)
			cmd.Dir = u.dir
			return cmd
		}
	}
	
	return nil
}