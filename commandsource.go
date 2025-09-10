package cmdrunner

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CommandInfo holds information about a command
type CommandInfo struct {
	Description string // Human-readable description
	Execution   string // What will actually be executed
}

// CommandSource represents a source of commands (mise, just, make, package.json, etc.)
type CommandSource interface {
	// Name returns the display name for this source (e.g., "mise", "npm", "Poetry")
	Name() string

	// ListCommands returns all available commands with description and execution command
	// Returns a map of command name to CommandInfo
	ListCommands() map[string]CommandInfo

	// FindCommand looks for a specific command in this source
	// Returns nil if the command is not found
	FindCommand(command string, args []string) *exec.Cmd

	// Priority returns the priority of this source (lower numbers = higher priority)
	// This determines the order in which sources are checked
	Priority() int
}

// Project represents a directory with multiple command sources
type Project struct {
	Dir            string
	CommandSources []CommandSource
}

// ResolveProject analyzes a directory and returns a Project with all applicable CommandSources
func ResolveProject(dir string) *Project {
	sources := []CommandSource{}

	// Check for command runners (highest priority)
	if FileExists(filepath.Join(dir, ".mise.toml")) {
		if source := NewMiseSource(dir); source != nil {
			sources = append(sources, source)
		}
	}

	if FileExists(filepath.Join(dir, "justfile")) || FileExists(filepath.Join(dir, "Justfile")) {
		if source := NewJustSource(dir); source != nil {
			sources = append(sources, source)
		}
	}

	if FileExists(filepath.Join(dir, "Makefile")) || FileExists(filepath.Join(dir, "makefile")) {
		if source := NewMakeSource(dir); source != nil {
			sources = append(sources, source)
		}
	}

	// Check for language-specific project files
	if FileExists(filepath.Join(dir, "package.json")) {
		if source := detectNodeProject(dir); source != nil {
			sources = append(sources, source)
		}
	}

	if FileExists(filepath.Join(dir, "pyproject.toml")) {
		if source := detectPythonProject(dir); source != nil {
			sources = append(sources, source)
		}
	}

	if FileExists(filepath.Join(dir, "Cargo.toml")) {
		if source := NewCargoSource(dir); source != nil {
			sources = append(sources, source)
		}
	}

	if FileExists(filepath.Join(dir, "go.mod")) {
		if source := NewGoSource(dir); source != nil {
			sources = append(sources, source)
		}
	}

	// Check for build tools
	if FileExists(filepath.Join(dir, "build.gradle")) || FileExists(filepath.Join(dir, "build.gradle.kts")) {
		if source := NewGradleSource(dir); source != nil {
			sources = append(sources, source)
		}
	}

	if FileExists(filepath.Join(dir, "pom.xml")) {
		if source := NewMavenSource(dir); source != nil {
			sources = append(sources, source)
		}
	}

	return &Project{
		Dir:            dir,
		CommandSources: sources,
	}
}

// detectNodeProject determines which Node.js package manager to use
func detectNodeProject(dir string) CommandSource {
	// Check for Deno first (as it can also have package.json)
	if FileExists(filepath.Join(dir, "deno.json")) || FileExists(filepath.Join(dir, "deno.jsonc")) {
		return NewDenoSource(dir)
	}

	// Check lockfiles to determine package manager
	if FileExists(filepath.Join(dir, "bun.lockb")) {
		return NewBunSource(dir)
	}

	if FileExists(filepath.Join(dir, "pnpm-lock.yaml")) {
		return NewPnpmSource(dir)
	}

	if FileExists(filepath.Join(dir, "yarn.lock")) {
		return NewYarnSource(dir)
	}

	if FileExists(filepath.Join(dir, "package-lock.json")) {
		return NewNpmSource(dir)
	}

	// Check for config files if no lockfile exists
	if FileExists(filepath.Join(dir, ".yarnrc.yml")) || FileExists(filepath.Join(dir, ".yarnrc")) {
		return NewYarnSource(dir)
	}

	// Default to npm
	return NewNpmSource(dir)
}

// detectPythonProject determines which Python package manager to use
func detectPythonProject(dir string) CommandSource {
	pyprojectPath := filepath.Join(dir, "pyproject.toml")

	// Check for Poetry
	if FileExists(filepath.Join(dir, "poetry.lock")) {
		return NewPoetrySource(dir)
	}

	// Check for uv
	if FileExists(filepath.Join(dir, "uv.lock")) || FileExists(filepath.Join(dir, ".uv")) {
		return NewUvSource(dir)
	}

	// Read pyproject.toml to determine the tool
	if data, err := os.ReadFile(pyprojectPath); err == nil {
		content := string(data)

		if strings.Contains(content, "[tool.poetry]") {
			return NewPoetrySource(dir)
		}

		if strings.Contains(content, "[tool.uv]") {
			return NewUvSource(dir)
		}
	}

	// Default to uv for modern Python projects with pyproject.toml
	return NewUvSource(dir)
}

// Base implementation helper for common functionality
type baseSource struct {
	dir      string
	name     string
	priority int
}

func (b *baseSource) Name() string {
	return b.name
}

func (b *baseSource) Priority() int {
	return b.priority
}

// Helper function to parse package.json scripts
func parsePackageJsonScripts(dir string) (map[string]string, error) {
	packageJSON := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(packageJSON)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	return pkg.Scripts, nil
}
