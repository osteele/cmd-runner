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
	// Special handling for 'check' command
	if r.Command == "check" {
		return HandleCheckCommand(r)
	}

	dirs := []string{r.CurrentDir}
	if r.ProjectRoot != r.CurrentDir {
		dirs = append(dirs, r.ProjectRoot)
	}

	for _, dir := range dirs {
		if cmd := r.FindCommand(dir); cmd != nil {
			return r.ExecuteCommand(cmd)
		}
	}

	return fmt.Errorf("no command '%s' found in current directory or project root", r.Command)
}

func (r *CommandRunner) FindCommand(dir string) *exec.Cmd {
	runners := []commandFinder{
		&miseRunner{},
		&justRunner{},
		&makeRunner{},
		&nodePackageRunner{},
		&cargoRunner{},
		&goRunner{},
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
		"format":    {"format", "fmt"},
		"run":       {"run", "dev", "serve", "start"},
		"dev":       {"dev", "run", "serve", "start"},
		"serve":     {"serve", "dev", "run", "start"},
		"build":     {"build"},
		"lint":      {"lint"},
		"test":      {"test", "tests"},
		"fix":       {"fix", "format-fix", "lint-fix"},
		"clean":     {"clean"},
		"install":   {"install", "setup"},
		"check":     {"check"},
		"typecheck": {"typecheck", "type-check", "types"},
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
		"run":       {"run", "dev", "serve", "start"},
		"dev":       {"dev", "run", "serve", "start"},
		"serve":     {"serve", "dev", "run", "start"},
		"start":     {"start", "run", "dev", "serve"},
		"build":     {"build"},
		"lint":      {"lint"},
		"test":      {"test"},
		"fix":       {"fix"},
		"clean":     {"clean"},
		"install":   {"install", "setup"},
		"setup":     {"setup", "install"},
		"check":     {"check"},
		"typecheck": {"typecheck"},
	}

	if alternatives, ok := aliases[cmd]; ok {
		return alternatives[0]
	}
	return cmd
}
