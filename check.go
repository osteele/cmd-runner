package cmdrunner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HandleCheckCommand handles the special 'check' command that runs lint, typecheck, and test
func HandleCheckCommand(r *CommandRunner) error {
	dirs := []string{r.CurrentDir}
	if r.ProjectRoot != r.CurrentDir {
		dirs = append(dirs, r.ProjectRoot)
	}

	// Try to find a native check command first
	for _, dir := range dirs {
		if cmd := r.findNativeCheckCommand(dir); cmd != nil {
			return r.ExecuteCommand(cmd)
		}
	}

	// If no native check command, run lint, typecheck, and test separately
	fmt.Fprintf(os.Stderr, "Running check (lint, typecheck, test)...\n")

	// Create runners for each sub-command
	commands := []string{"lint", "typecheck", "test"}
	for _, cmdName := range commands {
		fmt.Fprintf(os.Stderr, "\n→ Running %s...\n", cmdName)
		subRunner := &CommandRunner{
			Command:     cmdName,
			Args:        r.Args,
			CurrentDir:  r.CurrentDir,
			ProjectRoot: r.ProjectRoot,
		}

		if cmdName == "typecheck" {
			// For typecheck, if it doesn't exist as a command, skip it with a message
			if !r.hasTypecheckCommand() {
				fmt.Fprintf(os.Stderr, "  No separate typecheck command for this project type\n")
				continue
			}
		}

		if err := subRunner.Run(); err != nil {
			// If lint or typecheck fails, continue to run the other commands
			// but return the error at the end
			if cmdName != "test" {
				fmt.Fprintf(os.Stderr, "  Warning: %s failed: %v\n", cmdName, err)
				continue
			}
			return fmt.Errorf("%s failed: %w", cmdName, err)
		}
	}

	return nil
}

func (r *CommandRunner) findNativeCheckCommand(dir string) *exec.Cmd {
	// Check for mise
	if FileExists(filepath.Join(dir, ".mise.toml")) {
		testCmd := exec.Command("mise", "run", "--list")
		testCmd.Dir = dir
		if output, err := testCmd.Output(); err == nil && strings.Contains(string(output), "check") {
			return exec.Command("mise", append([]string{"run", "check"}, r.Args...)...)
		}
	}

	// Check for just
	if FileExists(filepath.Join(dir, "justfile")) || FileExists(filepath.Join(dir, "Justfile")) {
		testCmd := exec.Command("just", "--list")
		testCmd.Dir = dir
		if output, err := testCmd.Output(); err == nil && strings.Contains(string(output), "check") {
			return exec.Command("just", append([]string{"check"}, r.Args...)...)
		}
	}

	// Check for make
	if FileExists(filepath.Join(dir, "Makefile")) || FileExists(filepath.Join(dir, "makefile")) {
		makeRunner := &makeRunner{}
		if makeRunner.hasTarget(dir, "check") {
			return exec.Command("make", append([]string{"check"}, r.Args...)...)
		}
	}

	// Check for npm/yarn/bun scripts
	if FileExists(filepath.Join(dir, "package.json")) {
		data, err := os.ReadFile(filepath.Join(dir, "package.json"))
		if err == nil {
			var pkg struct {
				Scripts map[string]string `json:"scripts"`
			}
			if json.Unmarshal(data, &pkg) == nil {
				if _, ok := pkg.Scripts["check"]; ok {
					packageManager := detectPackageManager(dir)
					if packageManager != "" {
						return exec.Command(packageManager, append([]string{"run", "check"}, r.Args...)...)
					}
				}
			}
		}
	}

	return nil
}

func (r *CommandRunner) hasTypecheckCommand() bool {
	// Check if current project type supports typecheck
	dirs := []string{r.CurrentDir}
	if r.ProjectRoot != r.CurrentDir {
		dirs = append(dirs, r.ProjectRoot)
	}

	for _, dir := range dirs {
		// TypeScript projects
		if FileExists(filepath.Join(dir, "tsconfig.json")) {
			return true
		}

		// Python projects with pyright or mypy
		if FileExists(filepath.Join(dir, "pyproject.toml")) {
			data, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
			content := string(data)
			if strings.Contains(content, "pyright") || strings.Contains(content, "mypy") {
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
