package internal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/term"
)

type CommandRunner struct {
	Command     string
	Args        []string
	CurrentDir  string
	ProjectRoot string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	projects    []*Project // cached resolved projects
}

func New(command string, args []string) *CommandRunner {
	return &CommandRunner{
		Command: command, // Keep raw command; normalization happens in Run()
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

// resolveProjects returns cached resolved projects, resolving on first call.
func (r *CommandRunner) resolveProjects() []*Project {
	if r.projects != nil {
		return r.projects
	}
	r.projects = []*Project{ResolveProject(r.CurrentDir)}
	if r.ProjectRoot != r.CurrentDir && r.ProjectRoot != "" {
		r.projects = append(r.projects, ResolveProject(r.ProjectRoot))
	}
	return r.projects
}

func (r *CommandRunner) Run() error {
	// First, try to find the exact command (no normalization)
	if cmd := r.findExactCommand(r.Command, r.Args); cmd != nil {
		return r.ExecuteCommand(cmd)
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

	// Only after every source has rejected the exact name, try aliases in their
	// declared order. This lets a real command named "f", "run", etc. beat an
	// alias supplied by a higher-priority source.
	for _, variant := range GetCommandVariants(r.Command)[1:] {
		if cmd := r.findExactCommand(variant, r.Args); cmd != nil {
			return r.ExecuteCommand(cmd)
		}
	}

	return fmt.Errorf("no command '%s' found in current directory or project root", r.Command)
}

func (r *CommandRunner) findExactCommand(command string, args []string) *exec.Cmd {
	for _, project := range r.resolveProjects() {
		for _, source := range project.CommandSources {
			if cmd := source.FindCommand(command, args); cmd != nil {
				return cmd
			}
		}
	}
	return nil
}

func (r *CommandRunner) findCommand(command string, args []string) *exec.Cmd {
	for _, variant := range GetCommandVariants(command) {
		if cmd := r.findExactCommand(variant, args); cmd != nil {
			return cmd
		}
	}
	return nil
}

func (r *CommandRunner) ExecuteCommand(cmd *exec.Cmd) error {
	cmd.Stdin = r.stdinReader()
	cmd.Stdout = r.stdoutWriter()
	cmd.Stderr = r.stderrWriter()

	fmt.Fprintf(cmd.Stderr, "Running: %s\n", strings.Join(cmd.Args, " "))
	return cmd.Run()
}

func (r *CommandRunner) stdinReader() io.Reader {
	if r.Stdin != nil {
		return r.Stdin
	}
	return os.Stdin
}

func (r *CommandRunner) stdoutWriter() io.Writer {
	if r.Stdout != nil {
		return r.Stdout
	}
	return os.Stdout
}

func (r *CommandRunner) stderrWriter() io.Writer {
	if r.Stderr != nil {
		return r.Stderr
	}
	return os.Stderr
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

	projects := r.resolveProjects()

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
	synth := make(map[string]CommandInfo)
	if r.canSynthesizeCheck() {
		synth["check"] = CommandInfo{Description: "Runs lint, typecheck, and test", Execution: "synthesized"}
	}
	if r.canSynthesizeFix() {
		synth["fix"] = CommandInfo{Description: "Runs format and lint fix", Execution: "synthesized"}
	}
	if r.hasTypecheckCapability() {
		synth["typecheck"] = CommandInfo{Description: "Runs type checking", Execution: "synthesized"}
	}

	synthToShow := make(map[string]CommandInfo)
	for cmd, info := range synth {
		if shown[cmd] {
			continue
		}
		// Show synthesized typecheck only when there's no explicit one AND project supports it
		if cmd == "typecheck" {
			if hasExplicitTypecheck {
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
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // Default width
	}
	return width
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

// commandGroup defines a group of related commands.
// Canonical is the primary name. Variants lists other names to try (in order)
// when the canonical name isn't found. Names in variants that don't have their
// own commandGroup entry are treated as aliases that normalize to canonical.
type commandGroup struct {
	canonical string
	variants  []string
}

// commandGroups is the single source of truth for command aliasing.
// Both NormalizeCommand and GetCommandVariants derive from this.
var commandGroups = []commandGroup{
	{"format", []string{"fmt", "f"}},
	{"run", []string{"r", "dev", "serve", "start"}},
	{"dev", []string{"run", "serve", "start"}},
	{"serve", []string{"s", "dev", "run", "start"}},
	{"start", []string{"run", "dev", "serve"}},
	{"build", []string{"b"}},
	{"lint", []string{"l"}},
	{"test", []string{"t", "tests"}},
	{"typecheck", []string{"type-check", "types", "tc"}},
	{"fix", []string{"format-fix", "lint-fix"}},
	{"clean", nil},
	{"install", nil},
	{"setup", nil},
	{"check", nil},
}

// variantsMap and normalizeMap are built once from commandGroups at init.
var (
	variantsMap  map[string][]string
	normalizeMap map[string]string
)

func init() {
	variantsMap = make(map[string][]string)
	normalizeMap = make(map[string]string)

	// Collect canonical names so we can distinguish aliases from standalone commands
	canonicalSet := make(map[string]bool, len(commandGroups))
	for _, g := range commandGroups {
		canonicalSet[g.canonical] = true
	}

	for _, g := range commandGroups {
		// Full search list for this canonical command
		fullList := make([]string, 0, 1+len(g.variants))
		fullList = append(fullList, g.canonical)
		fullList = append(fullList, g.variants...)

		variantsMap[g.canonical] = fullList
		normalizeMap[g.canonical] = g.canonical

		// Build entries for each variant
		for _, v := range g.variants {
			if canonicalSet[v] {
				continue // has its own group, skip
			}
			// This variant is an alias — build its search list: [self, rest...]
			aliasVariants := make([]string, 0, len(fullList))
			aliasVariants = append(aliasVariants, v)
			for _, name := range fullList {
				if name != v {
					aliasVariants = append(aliasVariants, name)
				}
			}
			variantsMap[v] = aliasVariants
			normalizeMap[v] = g.canonical
		}
	}
}

func GetCommandVariants(command string) []string {
	if v, ok := variantsMap[command]; ok {
		return v
	}
	return []string{command}
}

func NormalizeCommand(cmd string) string {
	if normalized, ok := normalizeMap[cmd]; ok {
		return normalized
	}
	return cmd
}
