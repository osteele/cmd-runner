package cmdrunner

import (
	"bufio"
	"encoding/json"
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

func (r *CommandRunner) ListCommands() {
	dirs := []string{r.CurrentDir}
	if r.ProjectRoot != r.CurrentDir && r.ProjectRoot != "" {
		dirs = append(dirs, r.ProjectRoot)
	}

	fmt.Println("Available commands for this project:")
	fmt.Println()

	// Track what we've already shown to avoid duplicates
	shown := make(map[string]bool)

	// First show commands from current dir, then from project root
	for i, dir := range dirs {
		if i > 0 {
			relPath, _ := filepath.Rel(r.CurrentDir, dir)
			if relPath == "." {
				continue
			}
			fmt.Printf("\nFrom project root (%s):\n", relPath)
		}

		r.listCommandsInDir(dir, shown)
	}

	// Always show synthesized commands
	if !shown["check"] && !shown["lint"] && !shown["typecheck"] && !shown["test"] {
		fmt.Println("\nSynthesized commands (provided by cmd-runner):")
		if !shown["check"] {
			fmt.Println("  check      → Runs lint, typecheck, and test")
		}
		if !shown["fix"] {
			fmt.Println("  fix        → Runs format and lint fix")
		}
		if !shown["typecheck"] {
			fmt.Println("  typecheck  → Runs type checking")
		}
	}

	fmt.Println("\nCommand aliases:")
	fmt.Println("  f  → format     t  → test       tc → typecheck")
	fmt.Println("  r  → run        s  → serve      b  → build")
	fmt.Println("  l  → lint")
}

func (r *CommandRunner) listCommandsInDir(dir string, shown map[string]bool) {
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
		if commands := r.getRunnerCommands(runner, dir); len(commands) > 0 {
			runnerName := getRunnerName(runner)
			fmt.Printf("\n%s commands:\n", runnerName)
			for cmd, desc := range commands {
				if !shown[cmd] {
					fmt.Printf("  %-12s → %s\n", cmd, desc)
					shown[cmd] = true
				}
			}
		}
	}
}

func getRunnerName(runner commandFinder) string {
	switch runner.(type) {
	case *miseRunner:
		return "mise"
	case *justRunner:
		return "just"
	case *makeRunner:
		return "make"
	case *denoRunner:
		return "Deno"
	case *nodePackageRunner:
		return "Node.js"
	case *cargoRunner:
		return "Cargo"
	case *goRunner:
		return "Go"
	case *poetryRunner:
		return "Poetry"
	case *uvRunner:
		return "uv"
	case *gradleRunner:
		return "Gradle"
	case *mavenRunner:
		return "Maven"
	default:
		return "Unknown"
	}
}

func (r *CommandRunner) getRunnerCommands(runner commandFinder, dir string) map[string]string {
	commands := make(map[string]string)

	switch runner.(type) {
	case *miseRunner:
		if FileExists(filepath.Join(dir, ".mise.toml")) {
			// Try to list mise commands
			testCmd := exec.Command("mise", "run", "--list")
			testCmd.Dir = dir
			if output, err := testCmd.Output(); err == nil {
				r.parseMiseCommands(string(output), commands)
			}
		}
	case *justRunner:
		if FileExists(filepath.Join(dir, "justfile")) || FileExists(filepath.Join(dir, "Justfile")) {
			testCmd := exec.Command("just", "--list")
			testCmd.Dir = dir
			if output, err := testCmd.Output(); err == nil {
				r.parseJustCommands(string(output), commands)
			}
		}
	case *makeRunner:
		if FileExists(filepath.Join(dir, "Makefile")) || FileExists(filepath.Join(dir, "makefile")) {
			r.parseMakefileCommands(dir, commands)
		}
	case *nodePackageRunner:
		if FileExists(filepath.Join(dir, "package.json")) {
			r.parsePackageJsonCommands(dir, commands)
		}
	case *cargoRunner:
		if FileExists(filepath.Join(dir, "Cargo.toml")) {
			commands["build"] = "cargo build"
			commands["run"] = "cargo run"
			commands["test"] = "cargo test"
			commands["check"] = "cargo check"
			commands["format"] = "cargo fmt"
			commands["lint"] = "cargo clippy"
			commands["clean"] = "cargo clean"
		}
	case *goRunner:
		if FileExists(filepath.Join(dir, "go.mod")) {
			commands["build"] = "go build"
			commands["run"] = "go run ."
			commands["test"] = "go test ./..."
			commands["format"] = "go fmt ./..."
			commands["lint"] = "go vet ./..."
			commands["clean"] = "go clean"
		}
	case *poetryRunner:
		if FileExists(filepath.Join(dir, "poetry.lock")) || FileExists(filepath.Join(dir, "pyproject.toml")) {
			commands["install"] = "poetry install"
			commands["run"] = "poetry run python"
			commands["test"] = "poetry run pytest"
			commands["format"] = "poetry run ruff format"
			commands["lint"] = "poetry run ruff check"
			commands["typecheck"] = "poetry run pyright"
		}
	case *uvRunner:
		if FileExists(filepath.Join(dir, "pyproject.toml")) {
			commands["install"] = "uv sync"
			commands["run"] = "uv run"
			commands["test"] = "uv run pytest"
			commands["format"] = "uv run ruff format"
			commands["lint"] = "uv run ruff check"
			commands["typecheck"] = "uv run pyright"
		}
	}

	return commands
}

func (r *CommandRunner) parseMiseCommands(output string, commands map[string]string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Tasks:") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				cmd := parts[0]
				commands[cmd] = "mise run " + cmd
			}
		}
	}
}

func (r *CommandRunner) parseJustCommands(output string, commands map[string]string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Available") {
			// just output format: "command   # description"
			parts := strings.SplitN(line, " ", 2)
			if len(parts) > 0 {
				cmd := strings.TrimSpace(parts[0])
				if cmd != "" {
					commands[cmd] = "just " + cmd
				}
			}
		}
	}
}

func (r *CommandRunner) parseMakefileCommands(dir string, commands map[string]string) {
	makefiles := []string{"Makefile", "makefile"}
	for _, mf := range makefiles {
		path := filepath.Join(dir, mf)
		if FileExists(path) {
			file, err := os.Open(path)
			if err != nil {
				continue
			}
			defer func() {
				_ = file.Close()
			}()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				// Look for targets (lines ending with :)
				if strings.Contains(line, ":") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") {
					parts := strings.Split(line, ":")
					if len(parts) > 0 {
						target := strings.TrimSpace(parts[0])
						// Skip special targets and variables
						if !strings.HasPrefix(target, ".") && !strings.Contains(target, "=") && target != "" {
							commands[target] = "make " + target
						}
					}
				}
			}
		}
	}
}

func (r *CommandRunner) parsePackageJsonCommands(dir string, commands map[string]string) {
	packageJSON := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(packageJSON)
	if err != nil {
		return
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}

	packageManager := detectPackageManager(dir)
	if packageManager == "" {
		packageManager = "npm"
	}

	for script := range pkg.Scripts {
		if packageManager == "deno" {
			commands[script] = "deno task " + script
		} else {
			commands[script] = packageManager + " run " + script
		}
	}
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
		"f":         {"format"}, // Short alias for format
		"run":       {"run", "dev", "serve", "start"},
		"r":         {"run"}, // Short alias for run
		"dev":       {"dev", "run", "serve", "start"},
		"serve":     {"serve", "dev", "run", "start"},
		"s":         {"serve"}, // Short alias for serve/server
		"start":     {"start", "run", "dev", "serve"},
		"build":     {"build"},
		"b":         {"build"}, // Short alias for build
		"lint":      {"lint"},
		"l":         {"lint"}, // Short alias for lint
		"test":      {"test"},
		"t":         {"test"}, // Short alias for test
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
