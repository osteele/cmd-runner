package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unsafe"
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

// ListCommands is the original method for backward compatibility
func (r *CommandRunner) ListCommands() {
	r.ListCommandsWithOptions(false, false)
}

// ListCommandsWithOptions shows available commands with configurable options
func (r *CommandRunner) ListCommandsWithOptions(showAll bool, verbose bool) {
	fmt.Println("Available commands for this project:")
	fmt.Println()

	// Core commands we always want to show if they exist
	coreCommands := map[string]bool{
		"build": true, "check": true, "clean": true, "dev": true,
		"fix": true, "fmt": true, "format": true, "install": true,
		"lint": true, "run": true, "serve": true, "start": true,
		"test": true, "typecheck": true,
	}

	// Track what we've already shown to avoid duplicates
	shown := make(map[string]bool)

	// Build projects for current dir and project root
	projects := []*Project{}
	projects = append(projects, ResolveProject(r.CurrentDir))

	if r.ProjectRoot != r.CurrentDir && r.ProjectRoot != "" {
		projects = append(projects, ResolveProject(r.ProjectRoot))
	}

	sourcesShown := 0
	hasExplicitTypecheck := r.hasListedCommand("typecheck", "tc")

	// Show commands from each project
	for i, project := range projects {
		if i > 0 {
			relPath, _ := filepath.Rel(r.CurrentDir, project.Dir)
			if relPath == "." {
				continue
			}
			// If showing all sources, or if current dir had no commands, show project root
			if showAll || sourcesShown == 0 {
				fmt.Printf("\nFrom project root (%s):\n", relPath)
			} else {
				// Skip project root if we already showed commands from current dir
				continue
			}
		}

		// Show commands from each source
		for sourceIdx, source := range project.CommandSources {
			// If not showing all, only show the first source with commands
			if !showAll && sourcesShown > 0 {
				break
			}

			commands := source.ListCommands()
			if len(commands) == 0 {
				continue
			}

			// Separate core and additional commands
			core := make(map[string]CommandInfo)
			additional := make(map[string]CommandInfo)

			for cmd, info := range commands {
				if !shown[cmd] && !isPrivateCommand(cmd) {
					if coreCommands[cmd] {
						core[cmd] = info
					} else {
						additional[cmd] = info
					}
				}
			}

			// Only show source if it has commands
			if len(core) > 0 || len(additional) > 0 {
				fmt.Printf("\n%s commands:\n", source.Name())
				sourcesShown++

				// Show core commands first
				if len(core) > 0 {
					for _, cmd := range sortCommands(core) {
						if !shown[cmd] {
							r.printCommand(cmd, core[cmd], verbose)
							shown[cmd] = true
						}
					}
				}

				// Show additional commands
				if len(additional) > 0 {
					if len(core) > 0 {
						fmt.Println() // Add spacing between core and additional
					}
					for _, cmd := range sortCommands(additional) {
						if !shown[cmd] {
							r.printCommand(cmd, additional[cmd], verbose)
							shown[cmd] = true
						}
					}
				}
			}

			// If this is the primary source and we're not showing all, stop here
			if !showAll && sourceIdx == 0 && sourcesShown > 0 {
				break
			}
		}

		// If not showing all and we've shown a source, stop
		if !showAll && sourcesShown > 0 {
			break
		}
	}

	// Show synthesized commands if they're not already provided
	synth := map[string]CommandInfo{
		"check":     {Description: "Runs lint, typecheck, and test", Execution: "synthesized"},
		"fix":       {Description: "Runs format and lint fix", Execution: "synthesized"},
		"typecheck": {Description: "Runs type checking", Execution: "synthesized"},
	}

	synthToShow := make(map[string]CommandInfo)
	for cmd, info := range synth {
		if shown[cmd] {
			continue
		}
		// Show synthesized typecheck only when there's no explicit one AND project supports it
		if cmd == "typecheck" {
			if hasExplicitTypecheck || !r.hasTypecheckCapability() {
				continue
			}
		}
		synthToShow[cmd] = info
	}

	if len(synthToShow) > 0 {
		fmt.Println("\nSynthesized commands (provided by cmd-runner):")
		for _, cmd := range sortCommands(synthToShow) {
			r.printCommand(cmd, synthToShow[cmd], verbose)
		}
	}

	fmt.Println("\nCommand aliases:")
	fmt.Println("  f  → format     t  → test       tc → typecheck")
	fmt.Println("  r  → run        s  → serve      b  → build")
	fmt.Println("  l  → lint")
}

// getTerminalWidth returns the terminal width, defaulting to 80 if it can't be determined
func getTerminalWidth() int {
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	ws := &winsize{}
	retCode, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		return 80 // Default width
	}
	return int(ws.Col)
}

// printCommand prints a command with optional verbose description
func (r *CommandRunner) printCommand(cmd string, info CommandInfo, verbose bool) {
	if verbose {
		// Show both description and execution command
		fmt.Printf("  %-12s → %s\n", cmd, info.Description)
		fmt.Printf("  %-12s   (runs: %s)\n", "", info.Execution)
	} else {
		// Calculate available space for description
		termWidth := getTerminalWidth()
		// Account for: "  " (2) + command (12) + " → " (3) = 17 chars of overhead
		availableWidth := termWidth - 17
		if availableWidth < 20 {
			availableWidth = 20 // Minimum reasonable width
		}

		desc := info.Description
		if len(desc) > availableWidth {
			desc = desc[:availableWidth-3] + "..."
		}
		fmt.Printf("  %-12s → %s\n", cmd, desc)
	}
}

// sortCommands returns sorted command names from a map (works with any value type)
func sortCommands[T any](commands map[string]T) []string {
	keys := make([]string, 0, len(commands))
	for k := range commands {
		keys = append(keys, k)
	}

	// Alphabetically sort all commands
	sort.Strings(keys)

	return keys
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isPrivateCommand checks if a command should be hidden from listings
// Commands starting with underscore or dot are considered private/internal
func isPrivateCommand(name string) bool {
	return strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".")
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
		"install":   {"install"},
		"setup":     {"setup"},
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
		"install":   {"install"},
		"setup":     {"setup"},
		"check":     {"check"},
		"typecheck": {"typecheck"},
		"tc":        {"typecheck"}, // Short alias for typecheck
	}

	if alternatives, ok := aliases[cmd]; ok {
		return alternatives[0]
	}
	return cmd
}
