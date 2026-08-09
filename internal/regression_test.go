package internal

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type stubSource struct {
	name     string
	priority int
	commands map[string]*exec.Cmd
}

func (s *stubSource) Name() string {
	return s.name
}

func (s *stubSource) Priority() int {
	return s.priority
}

func (s *stubSource) ListCommands() map[string]CommandInfo {
	commands := make(map[string]CommandInfo, len(s.commands))
	for name := range s.commands {
		commands[name] = CommandInfo{Description: name, Execution: name}
	}
	return commands
}

func (s *stubSource) FindCommand(command string, _ []string) *exec.Cmd {
	return s.commands[command]
}

func TestRunPrefersExactCommandAcrossSources(t *testing.T) {
	var stdout bytes.Buffer
	runner := &CommandRunner{
		Command: "run",
		Stdout:  &stdout,
		Stderr:  io.Discard,
		projects: []*Project{{
			CommandSources: []CommandSource{
				&stubSource{name: "higher priority", priority: 1, commands: map[string]*exec.Cmd{
					"dev": exec.Command("printf", "alias"),
				}},
				&stubSource{name: "lower priority", priority: 10, commands: map[string]*exec.Cmd{
					"run": exec.Command("printf", "exact"),
				}},
			},
		}},
	}

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "exact" {
		t.Fatalf("Run() output = %q, want exact command output", got)
	}
}

func TestSynthesizedFixReturnsErrorWhenLintFails(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{"devDependencies":{"eslint":"latest"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &CommandRunner{
		CurrentDir:  dir,
		ProjectRoot: dir,
		Stdout:      io.Discard,
		Stderr:      io.Discard,
		projects: []*Project{{
			Dir: dir,
			CommandSources: []CommandSource{&stubSource{commands: map[string]*exec.Cmd{
				"format": exec.Command("true"),
				"lint":   exec.Command("false"),
			}}},
		}},
	}

	err := runner.synthesizeFixCommand()
	if err == nil || !strings.Contains(err.Error(), "lint --fix") {
		t.Fatalf("synthesizeFixCommand() error = %v, want lint failure", err)
	}
}

func TestSynthesizedCommandsRequireCapabilities(t *testing.T) {
	empty := &CommandRunner{projects: []*Project{{}}}
	if empty.canSynthesizeCheck() || empty.canSynthesizeFix() {
		t.Fatal("empty project should not advertise synthesized commands")
	}

	withTest := &CommandRunner{projects: []*Project{{
		CommandSources: []CommandSource{&stubSource{commands: map[string]*exec.Cmd{
			"test": exec.Command("true"),
		}}},
	}}}
	if !withTest.canSynthesizeCheck() {
		t.Fatal("project with tests should advertise synthesized check")
	}
}

func TestDenoProjectWithoutPackageJSON(t *testing.T) {
	dir := t.TempDir()
	config := `{"tasks":{"test":"echo test"}}`
	if err := os.WriteFile(filepath.Join(dir, "deno.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	project := ResolveProject(dir)
	source := findSourceByName(project.CommandSources, "Deno")
	if source == nil {
		t.Fatal("ResolveProject() did not detect deno.json without package.json")
	}
	cmd := source.FindCommand("test", nil)
	if cmd == nil || !reflect.DeepEqual(cmd.Args, []string{"deno", "task", "test"}) {
		t.Fatalf("Deno test command = %v, want deno task test", cmd)
	}
}

func TestPythonSourcesUseConfiguredTypechecker(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		new     func(string) CommandSource
		command []string
	}{
		{
			name:    "uv mypy",
			config:  "[tool.uv]\n\n[tool.mypy]\nstrict = true\n",
			new:     NewUvSource,
			command: []string{"uv", "run", "mypy", "."},
		},
		{
			name:    "poetry mypy",
			config:  "[tool.poetry]\nname = \"test\"\n\n[tool.mypy]\nstrict = true\n",
			new:     NewPoetrySource,
			command: []string{"poetry", "run", "mypy", "."},
		},
		{
			name:    "uv pyright",
			config:  "[tool.uv]\n\n[tool.pyright]\ntypeCheckingMode = \"strict\"\n",
			new:     NewUvSource,
			command: []string{"uv", "run", "pyright"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(tt.config), 0o644); err != nil {
				t.Fatal(err)
			}
			source := tt.new(dir)
			if source == nil {
				t.Fatal("source was not detected")
			}
			cmd := source.FindCommand("typecheck", nil)
			if cmd == nil || !reflect.DeepEqual(cmd.Args, tt.command) {
				t.Fatalf("typecheck command = %v, want %v", cmd, tt.command)
			}
		})
	}
}

func TestPythonSourceOmitsUnconfiguredTypecheck(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[tool.uv]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := NewUvSource(dir)
	if cmd := source.FindCommand("typecheck", nil); cmd != nil {
		t.Fatalf("unconfigured typecheck command = %v, want nil", cmd.Args)
	}
	if _, exists := source.ListCommands()["typecheck"]; exists {
		t.Fatal("unconfigured typecheck should not be listed")
	}
}

func TestExecuteCommandCanCaptureOutput(t *testing.T) {
	output := &synchronizedBuffer{}
	runner := &CommandRunner{
		Stdout: io.MultiWriter(io.Discard, output),
		Stderr: io.MultiWriter(io.Discard, output),
	}
	if err := runner.ExecuteCommand(exec.Command("sh", "-c", "printf stdout; printf stderr >&2")); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if !strings.Contains(got, "Running: sh -c") || !strings.Contains(got, "stdout") || !strings.Contains(got, "stderr") {
		t.Fatalf("captured output = %q", got)
	}
}
