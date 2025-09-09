package cmdrunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CommandRunner struct {
	Command     string
	Args        []string
	CurrentDir  string
	ProjectRoot string
}

func New(command string, args []string) *CommandRunner {
	return &CommandRunner{
		Command: NormalizeCommand(command),
		Args:    args,
	}
}

func (r *CommandRunner) Init() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	r.CurrentDir = cwd
	r.ProjectRoot = r.FindProjectRoot(cwd)
	return nil
}

func (r *CommandRunner) FindProjectRoot(dir string) string {
	current := dir
	for {
		if FileExists(filepath.Join(current, ".jj")) ||
			FileExists(filepath.Join(current, ".git")) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return dir
}

func (r *CommandRunner) Run() error {
	dirs := []string{r.CurrentDir}
	if r.ProjectRoot != r.CurrentDir {
		dirs = append(dirs, r.ProjectRoot)
	}

	// First try to find the command as-is (without normalization)
	// This allows actual commands named 'f', 't', etc. to take precedence
	// This also allows projects with their own 'check', 'fix', etc. to take precedence
	originalCommand := r.Command
	for _, dir := range dirs {
		if cmd := r.FindCommandExact(dir, originalCommand); cmd != nil {
			return r.ExecuteCommand(cmd)
		}
	}

	// Special handling for synthesized commands (only if no exact match found)
	switch r.Command {
	case "check":
		return HandleCheckCommand(r)
	case "fix":
		return HandleFixCommand(r)
	case "typecheck":
		return HandleTypecheckCommand(r)
	}

	// If no direct match found and the command might be an alias,
	// try with the normalized version
	for _, dir := range dirs {
		if cmd := r.FindCommand(dir); cmd != nil {
			return r.ExecuteCommand(cmd)
		}
	}

	return fmt.Errorf("no command '%s' found in current directory or project root", r.Command)
}

func (r *CommandRunner) FindCommandExact(dir string, command string) *exec.Cmd {
	runners := []commandFinder{
		&miseRunner{},
		&justRunner{},
		&makeRunner{},
		&denoRunner{},
		&nodePackageRunner{},
		&cargoRunner{},
		&goRunner{},
		&poetryRunner{},
		&uvRunner{},
		&gradleRunner{},
		&mavenRunner{},
	}

	// Try to find exact match - only use the command as-is, no variants
	for _, runner := range runners {
		// Use a special version that only checks for exact command match
		if cmd := findCommandExact(runner, dir, command, r.Args); cmd != nil {
			cmd.Dir = dir
			return cmd
		}
	}

	return nil
}

func (r *CommandRunner) FindCommand(dir string) *exec.Cmd {
	runners := []commandFinder{
		&miseRunner{},
		&justRunner{},
		&makeRunner{},
		&denoRunner{},
		&nodePackageRunner{},
		&cargoRunner{},
		&goRunner{},
		&poetryRunner{},
		&uvRunner{},
		&gradleRunner{},
		&mavenRunner{},
	}

	for _, runner := range runners {
		if cmd := runner.findCommand(dir, r.Command, r.Args); cmd != nil {
			cmd.Dir = dir
			return cmd
		}
	}

	return nil
}

func (r *CommandRunner) ExecuteCommand(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Fprintf(os.Stderr, "Running: %s\n", strings.Join(cmd.Args, " "))
	return cmd.Run()
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func GetCommandVariants(command string) []string {
	variants := map[string][]string{
		"format":    {"format", "fmt", "f"},
		"f":         {"f", "format", "fmt"},
		"run":       {"run", "r", "dev", "serve", "start"},
		"r":         {"r", "run", "dev", "serve", "start"},
		"dev":       {"dev", "run", "serve", "start"},
		"serve":     {"serve", "s", "dev", "run", "start"},
		"s":         {"s", "serve", "dev", "run", "start"},
		"build":     {"build", "b"},
		"b":         {"b", "build"},
		"lint":      {"lint", "l"},
		"l":         {"l", "lint"},
		"test":      {"test", "t", "tests"},
		"t":         {"t", "test", "tests"},
		"fix":       {"fix", "format-fix", "lint-fix"},
		"clean":     {"clean"},
		"install":   {"install", "setup"},
		"check":     {"check"},
		"typecheck": {"typecheck", "type-check", "types", "tc"},
		"tc":        {"tc", "typecheck", "type-check", "types"},
	}

	if v, ok := variants[command]; ok {
		return v
	}
	return []string{command}
}

func NormalizeCommand(cmd string) string {
	aliases := map[string][]string{
		"format":    {"format", "fmt"},
		"fmt":       {"format", "fmt"},
		"f":         {"format"},  // Short alias for format
		"run":       {"run", "dev", "serve", "start"},
		"r":         {"run"},      // Short alias for run
		"dev":       {"dev", "run", "serve", "start"},
		"serve":     {"serve", "dev", "run", "start"},
		"s":         {"serve"},    // Short alias for serve/server
		"start":     {"start", "run", "dev", "serve"},
		"build":     {"build"},
		"b":         {"build"},    // Short alias for build
		"lint":      {"lint"},
		"l":         {"lint"},     // Short alias for lint
		"test":      {"test"},
		"t":         {"test"},     // Short alias for test
		"fix":       {"fix"},
		"clean":     {"clean"},
		"install":   {"install", "setup"},
		"setup":     {"setup", "install"},
		"check":     {"check"},
		"typecheck": {"typecheck"},
		"tc":        {"typecheck"}, // Short alias for typecheck
	}

	if alternatives, ok := aliases[cmd]; ok {
		return alternatives[0]
	}
	return cmd
}
