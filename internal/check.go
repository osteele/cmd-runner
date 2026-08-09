package internal

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// HandleCheckCommand handles the special 'check' command that runs lint, typecheck, and test
func HandleCheckCommand(r *CommandRunner) error {
	// Try to find a native check command first
	for _, project := range r.resolveProjects() {
		if cmd := r.findNativeCheckCommand(project); cmd != nil {
			return r.ExecuteCommand(cmd)
		}
	}

	// If no native check command, synthesize by running lint, typecheck, and test separately
	return r.synthesizeCheckCommand()
}

// synthesizeCheckCommand runs lint, typecheck, and test as separate commands
func (r *CommandRunner) synthesizeCheckCommand() error {
	commands := []string{"lint", "typecheck", "test"}
	var foundAny bool
	var failedCommands []string
	var hasErrors bool

	// First check which commands are available
	for _, cmdName := range commands {
		if r.hasCommand(cmdName) {
			foundAny = true
		}
	}

	if !foundAny {
		return fmt.Errorf("no check, lint, typecheck, or test commands found")
	}

	fmt.Fprintf(r.stderrWriter(), "Running check (synthesizing from available commands)...\n")

	for _, cmdName := range commands {
		// Skip typecheck if it doesn't exist for this project type
		if cmdName == "typecheck" && !r.hasTypecheckCapability() {
			continue
		}

		if !r.hasCommand(cmdName) {
			continue
		}

		fmt.Fprintf(r.stderrWriter(), "\n→ Running %s...\n", cmdName)
		subRunner := &CommandRunner{
			Command:     cmdName,
			Args:        r.Args,
			CurrentDir:  r.CurrentDir,
			ProjectRoot: r.ProjectRoot,
			Stdin:       r.Stdin,
			Stdout:      r.Stdout,
			Stderr:      r.Stderr,
			projects:    r.resolveProjects(),
		}

		if err := subRunner.Run(); err != nil {
			hasErrors = true
			failedCommands = append(failedCommands, cmdName)
			fmt.Fprintf(r.stderrWriter(), "  ✗ %s failed: %v\n", cmdName, err)
		}
	}

	if hasErrors {
		return fmt.Errorf("check failed: %s", strings.Join(failedCommands, ", "))
	}

	return nil
}

func (r *CommandRunner) findNativeCheckCommand(project *Project) *exec.Cmd {
	// Look for "check" in command runner sources (mise, just, make)
	// These are highest priority for native check commands
	for _, source := range project.CommandSources {
		commands := source.ListCommands()
		if _, exists := commands["check"]; exists {
			if cmd := source.FindCommand("check", r.Args); cmd != nil {
				return cmd
			}
		}
	}
	return nil
}

// hasCommand checks if a command exists in any runner
func (r *CommandRunner) hasCommand(command string) bool {
	return r.findCommand(command, nil) != nil
}

func (r *CommandRunner) canSynthesizeCheck() bool {
	return r.hasCommand("lint") || r.hasCommand("typecheck") || r.hasCommand("test")
}

func (r *CommandRunner) canSynthesizeFix() bool {
	return r.hasCommand("format") || (r.supportsLintFix() && r.hasCommand("lint"))
}

// hasListedCommand reports whether any source explicitly lists one of the
// provided command names. This ignores synthesized fallbacks that don't appear
// in the source listings.
func (r *CommandRunner) hasListedCommand(names ...string) bool {
	for _, project := range r.resolveProjects() {
		for _, source := range project.CommandSources {
			commands := source.ListCommands()
			for _, name := range names {
				if _, ok := commands[name]; ok {
					return true
				}
			}
		}
	}

	return false
}

// hasTypecheckCapability checks if the project supports typechecking
func (r *CommandRunner) hasTypecheckCapability() bool {
	for _, project := range r.resolveProjects() {
		dir := project.Dir

		// TypeScript projects
		if FileExists(filepath.Join(dir, "tsconfig.json")) {
			return true
		}

		// Python projects with pyright or mypy
		if FileExists(filepath.Join(dir, "pyproject.toml")) {
			if pythonTypechecker(dir) != "" {
				return true
			}
		}

		// Rust always has cargo check
		if FileExists(filepath.Join(dir, "Cargo.toml")) {
			return true
		}

		// Go can use go build for type checking
		if FileExists(filepath.Join(dir, "go.mod")) {
			return true
		}
	}

	return false
}
