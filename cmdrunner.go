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
	// Build projects for current dir and project root
	projects := []*Project{}
	
	// Add current directory project
	projects = append(projects, ResolveProject(r.CurrentDir))
	
	// Add project root if different
	if r.ProjectRoot != r.CurrentDir && r.ProjectRoot != "" {
		projects = append(projects, ResolveProject(r.ProjectRoot))
	}
	
	// First, try to find the exact command (no normalization)
	for _, project := range projects {
		for _, source := range project.CommandSources {
			if cmd := source.FindCommand(r.Command, r.Args); cmd != nil {
				return r.ExecuteCommand(cmd)
			}
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
	normalizedCommand := NormalizeCommand(r.Command)
	if normalizedCommand != r.Command {
		for _, project := range projects {
			for _, source := range project.CommandSources {
				if cmd := source.FindCommand(normalizedCommand, r.Args); cmd != nil {
					return r.ExecuteCommand(cmd)
				}
			}
		}
	}
	
	return fmt.Errorf("no command '%s' found in current directory or project root", r.Command)
}


func (r *CommandRunner) ExecuteCommand(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Fprintf(os.Stderr, "Running: %s\n", strings.Join(cmd.Args, " "))
	return cmd.Run()
}

func (r *CommandRunner) ListCommands() {
	fmt.Println("Available commands for this project:")
	fmt.Println()

	// Track what we've already shown to avoid duplicates
	shown := make(map[string]bool)
	
	// Build projects for current dir and project root
	projects := []*Project{}
	projects = append(projects, ResolveProject(r.CurrentDir))
	
	if r.ProjectRoot != r.CurrentDir && r.ProjectRoot != "" {
		projects = append(projects, ResolveProject(r.ProjectRoot))
	}
	
	// Show commands from each project
	for i, project := range projects {
		if i > 0 {
			relPath, _ := filepath.Rel(r.CurrentDir, project.Dir)
			if relPath == "." {
				continue
			}
			fmt.Printf("\nFrom project root (%s):\n", relPath)
		}
		
		// Show commands from each source
		for _, source := range project.CommandSources {
			commands := source.ListCommands()
			if len(commands) > 0 {
				fmt.Printf("\n%s commands:\n", source.Name())
				for cmd, desc := range commands {
					if !shown[cmd] {
						fmt.Printf("  %-12s → %s\n", cmd, desc)
						shown[cmd] = true
					}
				}
			}
		}
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
