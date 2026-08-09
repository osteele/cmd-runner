package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCommandSourcesBuildExpectedCommands(t *testing.T) {
	fakeBin := t.TempDir()
	writeFixture(t, fakeBin, "mise", "#!/bin/sh\nprintf 'test  Run tests\\n'\n", 0o755)
	writeFixture(t, fakeBin, "just", "#!/bin/sh\nprintf 'Available recipes:\\ntest # Run tests\\n'\n", 0o755)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	tests := []struct {
		name     string
		files    map[string]string
		new      func(string) CommandSource
		command  string
		args     []string
		wantArgs []string
	}{
		{
			name:     "mise",
			files:    map[string]string{".mise.toml": "[tasks.test]\nrun = \"go test ./...\"\n"},
			new:      NewMiseSource,
			command:  "test",
			wantArgs: []string{"mise", "run", "test"},
		},
		{
			name:     "just",
			files:    map[string]string{"justfile": "test:\n\tgo test ./...\n"},
			new:      NewJustSource,
			command:  "test",
			wantArgs: []string{"just", "test"},
		},
		{
			name:     "make",
			files:    map[string]string{"Makefile": "test:\n\tgo test ./...\n"},
			new:      NewMakeSource,
			command:  "test",
			wantArgs: []string{"make", "test"},
		},
		{
			name:     "npm",
			files:    map[string]string{"package.json": `{"scripts":{"test":"vitest"}}`},
			new:      NewNpmSource,
			command:  "test",
			args:     []string{"--watch"},
			wantArgs: []string{"npm", "run", "test", "--", "--watch"},
		},
		{
			name:     "bun",
			files:    map[string]string{"package.json": `{"scripts":{"test":"bun test"}}`},
			new:      NewBunSource,
			command:  "test",
			wantArgs: []string{"bun", "run", "test"},
		},
		{
			name:     "pnpm",
			files:    map[string]string{"package.json": `{"scripts":{"test":"vitest"}}`},
			new:      NewPnpmSource,
			command:  "test",
			wantArgs: []string{"pnpm", "run", "test"},
		},
		{
			name:     "yarn",
			files:    map[string]string{"package.json": `{"scripts":{"test":"vitest"}}`},
			new:      NewYarnSource,
			command:  "test",
			wantArgs: []string{"yarn", "run", "test"},
		},
		{
			name:     "deno JSONC",
			files:    map[string]string{"deno.jsonc": "{\n  // project tasks\n  \"tasks\": {\n    \"test\": \"deno test\",\n  },\n}\n"},
			new:      NewDenoSource,
			command:  "test",
			wantArgs: []string{"deno", "task", "test"},
		},
		{
			name:     "poetry",
			files:    map[string]string{"pyproject.toml": "[tool.poetry]\nname = \"fixture\"\n"},
			new:      NewPoetrySource,
			command:  "setup",
			wantArgs: []string{"poetry", "install"},
		},
		{
			name:     "uv",
			files:    map[string]string{"pyproject.toml": "[tool.uv]\n"},
			new:      NewUvSource,
			command:  "setup",
			wantArgs: []string{"uv", "sync"},
		},
		{
			name:     "cargo binary",
			files:    map[string]string{"Cargo.toml": "[package]\nname = \"fixture\"\nversion = \"0.1.0\"\n\n[[bin]]\nname = \"worker\"\npath = \"src/worker.rs\"\n"},
			new:      NewCargoSource,
			command:  "run:worker",
			wantArgs: []string{"cargo", "run", "--bin", "worker"},
		},
		{
			name:     "gradle wrapper",
			files:    map[string]string{"build.gradle": "plugins {}\n", "gradlew": "#!/bin/sh\n"},
			new:      NewGradleSource,
			command:  "test",
			wantArgs: []string{"./gradlew", "test"},
		},
		{
			name:     "maven wrapper",
			files:    map[string]string{"pom.xml": "<project/>\n", "mvnw": "#!/bin/sh\n"},
			new:      NewMavenSource,
			command:  "test",
			wantArgs: []string{"./mvnw", "test"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range test.files {
				writeFixture(t, dir, name, content, 0o644)
			}
			source := test.new(dir)
			if source == nil {
				t.Fatal("source was not detected")
			}
			command := source.FindCommand(test.command, test.args)
			if command == nil {
				t.Fatalf("FindCommand(%q) returned nil", test.command)
			}
			if !reflect.DeepEqual(command.Args, test.wantArgs) {
				t.Fatalf("command args = %v, want %v", command.Args, test.wantArgs)
			}
			if _, exists := source.ListCommands()[test.command]; !exists {
				t.Fatalf("ListCommands() does not contain %q", test.command)
			}
		})
	}
}

func TestGoSourceDiscoversCommandLayout(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "go.mod", "module example.com/fixture\n\ngo 1.25.0\n", 0o644)
	writeFixture(t, dir, "library.go", "package fixture\n", 0o644)
	writeFixture(t, dir, "cmd/tool/main.go", "package main\nfunc main() {}\n", 0o644)

	source := NewGoSource(dir)
	runCommand := source.FindCommand("run", []string{"argument"})
	if runCommand == nil || !reflect.DeepEqual(runCommand.Args, []string{"go", "run", "./cmd/tool", "argument"}) {
		t.Fatalf("run command = %v, want go run ./cmd/tool argument", runCommand)
	}
	installCommand := source.FindCommand("install", nil)
	if installCommand == nil || !reflect.DeepEqual(installCommand.Args, []string{"go", "install", "./cmd/tool"}) {
		t.Fatalf("install command = %v, want go install ./cmd/tool", installCommand)
	}
	if got := source.ListCommands()["run"].Execution; got != "go run ./cmd/tool" {
		t.Fatalf("listed run command = %q", got)
	}
}

func TestGoSourceNamesMultipleMainPackages(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "go.mod", "module example.com/fixture\n\ngo 1.25.0\n", 0o644)
	writeFixture(t, dir, "cmd/server/main.go", "package main\nfunc main() {}\n", 0o644)
	writeFixture(t, dir, "cmd/worker/main.go", "package main\nfunc main() {}\n", 0o644)

	commands := NewGoSource(dir).ListCommands()
	if _, exists := commands["run"]; exists {
		t.Fatal("ambiguous Go project should not list a bare run command")
	}
	for _, command := range []string{"run:server", "run:worker"} {
		if _, exists := commands[command]; !exists {
			t.Errorf("Go commands do not contain %q: %v", command, commands)
		}
	}
}

func TestGoSourceCachesAndInvalidatesDiscovery(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "go.mod", "module example.com/fixture\n\ngo 1.25.0\n", 0o644)
	writeFixture(t, dir, "main.go", "package main\nfunc main() {}\n", 0o644)

	source := NewGoSource(dir).(*GoSource)
	discoveryCalls := 0
	source.discoverProject = func(dir string) goProjectDiscovery {
		discoveryCalls++
		return discoverGoProject(dir)
	}
	source.ListCommands()
	source.ListCommands()
	if discoveryCalls != 1 {
		t.Fatalf("unchanged project discovered %d times, want 1", discoveryCalls)
	}

	writeFixture(t, dir, "new.go", "package main\nvar changed = true\n", 0o644)
	source.ListCommands()
	if discoveryCalls != 2 {
		t.Fatalf("changed project discovered %d times, want 2", discoveryCalls)
	}
}

func TestGoWorkspaceDiscoversNestedModules(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "go.work", "go 1.25.0\n\nuse (\n\t./services/api\n\t./tools\n)\n", 0o644)
	writeFixture(t, dir, "services/api/go.mod", "module example.com/api\n\ngo 1.25.0\n", 0o644)
	writeFixture(t, dir, "services/api/cmd/server/main.go", "package main\nfunc main() {}\n", 0o644)
	writeFixture(t, dir, "tools/go.mod", "module example.com/tools\n\ngo 1.25.0\n", 0o644)
	writeFixture(t, dir, "tools/cmd/generate/main.go", "package main\nfunc main() {}\n", 0o644)

	project := ResolveProject(dir)
	source := findSourceByName(project.CommandSources, "Go")
	if source == nil {
		t.Fatal("Go workspace was not detected")
	}
	commands := source.ListCommands()
	for _, command := range []string{"run:server", "run:generate"} {
		if _, exists := commands[command]; !exists {
			t.Errorf("workspace commands do not contain %q: %v", command, commands)
		}
	}
	if got := commands["build"].Execution; got != "go build ./services/api/... ./tools/..." {
		t.Fatalf("build execution = %q", got)
	}
	if got := commands["setup"].Execution; got != "go work sync" {
		t.Fatalf("setup execution = %q", got)
	}
}

func TestCargoSourceDiscoversImplicitBinaries(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "Cargo.toml", "[package]\nname = \"fixture\"\nversion = \"0.1.0\"\n", 0o644)
	writeFixture(t, dir, "src/bin/server.rs", "fn main() {}\n", 0o644)
	writeFixture(t, dir, "src/bin/worker/main.rs", "fn main() {}\n", 0o644)

	source := NewCargoSource(dir)
	for _, binaryName := range []string{"server", "worker"} {
		commandName := "run:" + binaryName
		if _, exists := source.ListCommands()[commandName]; !exists {
			t.Errorf("Cargo commands do not contain %q", commandName)
		}
		command := source.FindCommand(commandName, nil)
		if command == nil || !reflect.DeepEqual(command.Args, []string{"cargo", "run", "--bin", binaryName}) {
			t.Errorf("%s command = %v", commandName, command)
		}
	}
}

func TestCargoSourceRespectsDisabledAutomaticBinaries(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "Cargo.toml", "[package]\nname = \"fixture\"\nversion = \"0.1.0\"\nautobins = false\n", 0o644)
	writeFixture(t, dir, "src/bin/server.rs", "fn main() {}\n", 0o644)

	if command := NewCargoSource(dir).FindCommand("run:server", nil); command != nil {
		t.Fatalf("disabled automatic binary produced command %v", command.Args)
	}
}

func TestStructuredMetadataIgnoresCommentsAndUnrelatedStrings(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "pyproject.toml", "# [tool.poetry]\n[project]\nname = \"mentions-mypy\"\ndependencies = [\"requests\"]\n", 0o644)
	if source := NewPoetrySource(dir); source != nil {
		t.Fatalf("commented Poetry table detected as %T", source)
	}
	if checker := pythonTypechecker(dir); checker != "" {
		t.Fatalf("unrelated project name selected type checker %q", checker)
	}
	poetryDir := t.TempDir()
	writeFixture(t, poetryDir, "pyproject.toml", "[tool.poetry]\nname = \"mentions-mypy\"\n", 0o644)
	if checker := pythonTypechecker(poetryDir); checker != "" {
		t.Fatalf("Poetry project name selected type checker %q", checker)
	}

	writeFixture(t, dir, "Cargo.toml", "[package]\nname = \"fixture\"\nversion = \"0.1.0\"\n# name = \"ghost\"\n", 0o644)
	if command := NewCargoSource(dir).FindCommand("run:ghost", nil); command != nil {
		t.Fatalf("commented Cargo binary produced command %v", command.Args)
	}
}

func TestPythonDependencyGroupsSelectTypechecker(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "pyproject.toml", "[tool.uv]\n\n[dependency-groups]\ndev = [\"mypy>=1.15\"]\n", 0o644)
	if checker := pythonTypechecker(dir); checker != "mypy" {
		t.Fatalf("pythonTypechecker() = %q, want mypy", checker)
	}
}

func TestPoetryDependencyGroupSelectsTypechecker(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "pyproject.toml", "[tool.poetry]\nname = \"fixture\"\n\n[tool.poetry.group.dev.dependencies]\npyright = \"^1.1\"\n", 0o644)
	if checker := pythonTypechecker(dir); checker != "pyright" {
		t.Fatalf("pythonTypechecker() = %q, want pyright", checker)
	}
}

func TestJSONCStandardizationPreservesCommentMarkersInStrings(t *testing.T) {
	input := []byte(`{"url":"https://example.com/a/*b*/",/* comment */"items":[1,2,],}`)
	standardJSON, err := standardizeJSONC(input)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(standardJSON); !strings.Contains(got, `"https://example.com/a/*b*/"`) {
		t.Fatalf("standard JSON changed string contents: %s", got)
	}
	var result struct {
		Items []int `json:"items"`
	}
	if err := json.Unmarshal(standardJSON, &result); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Items, []int{1, 2}) {
		t.Fatalf("items = %v, want [1 2]", result.Items)
	}
}

func writeFixture(t *testing.T, dir, name, content string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}
