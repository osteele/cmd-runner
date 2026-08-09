package internal

import (
	"os/exec"
	"path/filepath"
)

// PoetrySource for Poetry projects
type PoetrySource struct {
	baseSource
}

func NewPoetrySource(dir string) CommandSource {
	// Verify it's actually a Poetry project
	if !FileExists(filepath.Join(dir, "poetry.lock")) {
		config, err := readPythonProjectConfig(dir)
		if err != nil || !config.hasTool("poetry") {
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

func (p *PoetrySource) ListCommands() map[string]CommandInfo {
	commands := map[string]CommandInfo{
		"setup":   {Description: "Install dependencies for development", Execution: "poetry install"},
		"install": {Description: "Install package globally", Execution: "pip install ."},
		"run":     {Description: "Run Python interpreter", Execution: "poetry run python"},
		"test":    {Description: "Run tests", Execution: "poetry run pytest"},
		"format":  {Description: "Format code", Execution: "poetry run ruff format"},
		"lint":    {Description: "Run linter", Execution: "poetry run ruff check"},
		"build":   {Description: "Build distribution", Execution: "poetry build"},
		"publish": {Description: "Publish to PyPI", Execution: "poetry publish"},
	}
	if checker := pythonTypechecker(p.dir); checker != "" {
		commands["typecheck"] = CommandInfo{
			Description: "Run type checker",
			Execution:   "poetry run " + checker,
		}
	}
	return commands
}

func (p *PoetrySource) FindCommand(command string, args []string) *exec.Cmd {
	poetryCommands := map[string][]string{
		"setup":   {"install"},
		"run":     {"run", "python"},
		"test":    {"run", "pytest"},
		"lint":    {"run", "ruff", "check"},
		"format":  {"run", "ruff", "format"},
		"build":   {"build"},
		"publish": {"publish"},
	}

	// Check for install first (before variant matching)
	// This ensures "install" doesn't accidentally match "setup" as a variant
	if command == "install" {
		cmdArgs := append([]string{"install", "."}, args...)
		cmd := exec.Command("pip", cmdArgs...)
		cmd.Dir = p.dir
		return cmd
	}

	if command == "typecheck" {
		if checker := pythonTypechecker(p.dir); checker != "" {
			cmdArgs := pythonTypecheckArgs([]string{"run"}, checker, args)
			cmd := exec.Command("poetry", cmdArgs...)
			cmd.Dir = p.dir
			return cmd
		}
	}

	if poetryCmd, ok := poetryCommands[command]; ok {
		cmdArgs := make([]string, len(poetryCmd), len(poetryCmd)+len(args))
		copy(cmdArgs, poetryCmd)
		cmdArgs = append(cmdArgs, args...)
		cmd := exec.Command("poetry", cmdArgs...)
		cmd.Dir = p.dir
		return cmd
	}

	return nil
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
	} else if config, err := readPythonProjectConfig(dir); err == nil {
		if config.hasTool("uv") {
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

func (u *UvSource) ListCommands() map[string]CommandInfo {
	commands := map[string]CommandInfo{
		"setup":   {Description: "Install dependencies for development", Execution: "uv sync"},
		"install": {Description: "Install tool globally", Execution: "uv tool install ."},
		"run":     {Description: "Run a command", Execution: "uv run"},
		"test":    {Description: "Run tests", Execution: "uv run pytest"},
		"format":  {Description: "Format code", Execution: "uv run ruff format"},
		"lint":    {Description: "Run linter", Execution: "uv run ruff check"},
	}
	if checker := pythonTypechecker(u.dir); checker != "" {
		commands["typecheck"] = CommandInfo{
			Description: "Run type checker",
			Execution:   "uv run " + checker,
		}
	}
	return commands
}

func (u *UvSource) FindCommand(command string, args []string) *exec.Cmd {
	uvCommands := map[string][]string{
		"setup":   {"sync"},
		"install": {"tool", "install", "."},
		"run":     {"run"},
		"test":    {"run", "pytest"},
		"lint":    {"run", "ruff", "check"},
		"format":  {"run", "ruff", "format"},
	}

	if command == "typecheck" {
		if checker := pythonTypechecker(u.dir); checker != "" {
			cmdArgs := pythonTypecheckArgs([]string{"run"}, checker, args)
			cmd := exec.Command("uv", cmdArgs...)
			cmd.Dir = u.dir
			return cmd
		}
	}

	if uvCmd, ok := uvCommands[command]; ok {
		cmdArgs := make([]string, len(uvCmd), len(uvCmd)+len(args))
		copy(cmdArgs, uvCmd)
		cmdArgs = append(cmdArgs, args...)
		cmd := exec.Command("uv", cmdArgs...)
		cmd.Dir = u.dir
		return cmd
	}

	return nil
}

func pythonTypechecker(dir string) string {
	config, err := readPythonProjectConfig(dir)
	if err != nil {
		return ""
	}
	if config.hasTool("pyright") || config.hasDependency("pyright") {
		return "pyright"
	}
	if config.hasTool("mypy") || config.hasDependency("mypy") {
		return "mypy"
	}
	return ""
}

func pythonTypecheckArgs(prefix []string, checker string, args []string) []string {
	cmdArgs := append([]string{}, prefix...)
	cmdArgs = append(cmdArgs, checker)
	if checker == "mypy" {
		cmdArgs = append(cmdArgs, ".")
	}
	return append(cmdArgs, args...)
}
