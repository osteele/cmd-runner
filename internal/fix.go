package internal

import (
	"fmt"
	"path/filepath"
	"strings"
)

// HandleFixCommand handles the special 'fix' command that runs format/lint fixes
func HandleFixCommand(r *CommandRunner) error {
	// Try to find a native fix command first
	for _, project := range r.resolveProjects() {
		for _, source := range project.CommandSources {
			if cmd := source.FindCommand("fix", r.Args); cmd != nil {
				return r.ExecuteCommand(cmd)
			}
		}
	}

	// If no native fix command, synthesize by running format and lint fix commands
	return r.synthesizeFixCommand()
}

// synthesizeFixCommand runs format and lint fix commands
func (r *CommandRunner) synthesizeFixCommand() error {
	// Commands to try for fixing, in order of preference
	// Note: We'll deduplicate format/fmt below
	fixCommands := []struct {
		command string
		args    []string
	}{
		{"format", nil},
		{"fmt", nil},
	}

	// Add lint --fix only for projects that support it
	if r.supportsLintFix() {
		fixCommands = append(fixCommands, struct {
			command string
			args    []string
		}{"lint", []string{"--fix"}})
	}

	var foundAny bool
	var failedCommands []string

	// First check if any fix-related commands are available
	for _, fc := range fixCommands {
		if r.hasCommand(fc.command) {
			foundAny = true
		}
	}

	if !foundAny {
		return fmt.Errorf("no fix, format, or lint commands found")
	}

	fmt.Fprintf(r.stderrWriter(), "Running fix (synthesizing from available commands)...\n")

	// Track what we've already run to avoid duplicates
	executedTypes := make(map[string]bool)

	for _, fc := range fixCommands {
		if !r.hasCommand(fc.command) {
			continue
		}

		// Skip if we already ran a formatting command (format and fmt are equivalent)
		if (fc.command == "format" || fc.command == "fmt") && executedTypes["format"] {
			continue
		}

		cmdDisplay := fc.command
		if len(fc.args) > 0 {
			cmdDisplay = fmt.Sprintf("%s %s", fc.command, strings.Join(fc.args, " "))
		}

		fmt.Fprintf(r.stderrWriter(), "\n→ Running %s...\n", cmdDisplay)

		tempRunner := &CommandRunner{
			Command:     fc.command,
			Args:        append(fc.args, r.Args...),
			CurrentDir:  r.CurrentDir,
			ProjectRoot: r.ProjectRoot,
			Stdin:       r.Stdin,
			Stdout:      r.Stdout,
			Stderr:      r.Stderr,
			projects:    r.resolveProjects(),
		}

		if err := tempRunner.Run(); err != nil {
			// For fix commands, we often want to continue even if one fails
			failedCommands = append(failedCommands, cmdDisplay)
			fmt.Fprintf(r.stderrWriter(), "  ✗ %s failed: %v\n", cmdDisplay, err)
		} else {
			// Mark format as executed for both format and fmt commands
			if fc.command == "format" || fc.command == "fmt" {
				executedTypes["format"] = true
			}
		}
	}

	if len(failedCommands) > 0 {
		return fmt.Errorf("fix failed: %s", strings.Join(failedCommands, ", "))
	}

	return nil
}

// supportsLintFix checks if the project's lint command supports a --fix flag
func (r *CommandRunner) supportsLintFix() bool {
	for _, project := range r.resolveProjects() {
		dir := project.Dir

		// Go projects don't support lint --fix (go vet has no --fix flag)
		if FileExists(filepath.Join(dir, "go.mod")) {
			return false
		}

		// Node.js projects with ESLint typically support --fix
		if FileExists(filepath.Join(dir, "package.json")) {
			if packageConfig, err := readNodePackage(dir); err == nil && packageConfig.hasDependency("eslint") {
				return true
			}
		}

		// Python projects with ruff support --fix
		if FileExists(filepath.Join(dir, "pyproject.toml")) {
			if config, err := readPythonProjectConfig(dir); err == nil &&
				(config.hasTool("ruff") || config.hasDependency("ruff")) {
				return true
			}
		}

		// Rust clippy supports --fix
		if FileExists(filepath.Join(dir, "Cargo.toml")) {
			return false
		}
	}

	return false
}
