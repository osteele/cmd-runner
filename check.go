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

	fmt.Fprintf(os.Stderr, "Running check (synthesizing from available commands)...\n")

	for _, cmdName := range commands {
		// Skip typecheck if it doesn't exist for this project type
		if cmdName == "typecheck" && !r.hasTypecheckCapability() {
			continue
		}

		if !r.hasCommand(cmdName) {
			continue
		}

		fmt.Fprintf(os.Stderr, "\n→ Running %s...\n", cmdName)
		subRunner := &CommandRunner{
			Command:     cmdName,
			Args:        r.Args,
			CurrentDir:  r.CurrentDir,
			ProjectRoot: r.ProjectRoot,
		}

		if err := subRunner.Run(); err != nil {
			hasErrors = true
			failedCommands = append(failedCommands, cmdName)
			fmt.Fprintf(os.Stderr, "  ✗ %s failed: %v\n", cmdName, err)
		}
	}

	if hasErrors {
		return fmt.Errorf("check failed: %s", strings.Join(failedCommands, ", "))
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

// hasCommand checks if a command exists in any runner
func (r *CommandRunner) hasCommand(command string) bool {
	dirs := []string{r.CurrentDir}
	if r.ProjectRoot != r.CurrentDir {
		dirs = append(dirs, r.ProjectRoot)
	}

	// Create a temporary runner to check for the specific command
	// Build projects and check if command exists
	projects := []*Project{}
	projects = append(projects, ResolveProject(r.CurrentDir))
	if r.ProjectRoot != r.CurrentDir && r.ProjectRoot != "" {
		projects = append(projects, ResolveProject(r.ProjectRoot))
	}
	
	for _, project := range projects {
		for _, source := range project.CommandSources {
			if cmd := source.FindCommand(command, []string{}); cmd != nil {
				return true
			}
		}
	}
	return false
}

// hasTypecheckCapability checks if the project supports typechecking
func (r *CommandRunner) hasTypecheckCapability() bool {
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
