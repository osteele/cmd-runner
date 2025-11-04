package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CargoSource for Rust projects
type CargoSource struct {
	baseSource
}

func NewCargoSource(dir string) CommandSource {
	if !FileExists(filepath.Join(dir, "Cargo.toml")) {
		return nil
	}

	return &CargoSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "Cargo",
			priority: 10,
		},
	}
}

func (c *CargoSource) ListCommands() map[string]CommandInfo {
	return map[string]CommandInfo{
		"build":   {Description: "Build the project", Execution: "cargo build"},
		"run":     {Description: "Run the project", Execution: "cargo run"},
		"test":    {Description: "Run tests", Execution: "cargo test"},
		"check":   {Description: "Check code for errors", Execution: "cargo check"},
		"format":  {Description: "Format code", Execution: "cargo fmt"},
		"lint":    {Description: "Run clippy linter", Execution: "cargo clippy"},
		"clean":   {Description: "Clean build artifacts", Execution: "cargo clean"},
		"setup":   {Description: "Download dependencies", Execution: "cargo fetch"},
		"install": {Description: "Install binary globally", Execution: "cargo install --path ."},
	}
}

func (c *CargoSource) FindCommand(command string, args []string) *exec.Cmd {
	cargoCommands := map[string]string{
		"build":     "build",
		"run":       "run",
		"test":      "test",
		"lint":      "clippy",
		"format":    "fmt",
		"fmt":       "fmt",
		"clean":     "clean",
		"typecheck": "check",
		"tc":        "check",
		"check":     "check",
		"fix":       "fix",
		"setup":     "fetch",
		"install":   "install",
		"publish":   "publish",
	}

	for _, variant := range GetCommandVariants(command) {
		if cargoCmd, ok := cargoCommands[variant]; ok {
			var cmdArgs []string
			if cargoCmd == "install" {
				// Modern cargo requires --path for installing from current directory
				cmdArgs = append([]string{"install", "--path", "."}, args...)
			} else {
				cmdArgs = append([]string{cargoCmd}, args...)
			}
			cmd := exec.Command("cargo", cmdArgs...)
			cmd.Dir = c.dir
			return cmd
		}
	}

	// Try to handle custom binary targets
	cargoToml := filepath.Join(c.dir, "Cargo.toml")
	if data, err := os.ReadFile(cargoToml); err == nil {
		content := string(data)

		// Check for binary targets (run:binary-name pattern)
		if strings.HasPrefix(command, "run:") {
			binName := strings.TrimPrefix(command, "run:")
			if strings.Contains(content, `name = "`+binName+`"`) {
				cmdArgs := append([]string{"run", "--bin", binName}, args...)
				cmd := exec.Command("cargo", cmdArgs...)
				cmd.Dir = c.dir
				return cmd
			}
		}
	}

	return nil
}

// GoSource for Go projects
type GoSource struct {
	baseSource
}

func NewGoSource(dir string) CommandSource {
	if !FileExists(filepath.Join(dir, "go.mod")) {
		return nil
	}

	return &GoSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "Go",
			priority: 10,
		},
	}
}

func (g *GoSource) ListCommands() map[string]CommandInfo {
	return map[string]CommandInfo{
		"build":   {Description: "Build the project", Execution: "go build"},
		"run":     {Description: "Run the project", Execution: "go run ."},
		"test":    {Description: "Run tests", Execution: "go test ./..."},
		"format":  {Description: "Format code", Execution: "go fmt ./..."},
		"lint":    {Description: "Run linter", Execution: "go vet ./..."},
		"clean":   {Description: "Clean build artifacts", Execution: "go clean"},
		"setup":   {Description: "Download dependencies", Execution: "go mod download"},
		"install": {Description: "Install binary globally", Execution: "go install ."},
	}
}

func (g *GoSource) FindCommand(command string, args []string) *exec.Cmd {
	goCommands := map[string][]string{
		"build":     {"build"},
		"run":       {"run", "."},
		"test":      {"test", "./..."},
		"format":    {"fmt", "./..."},
		"fmt":       {"fmt", "./..."},
		"clean":     {"clean"},
		"setup":     {"mod", "download"},
		"install":   {"install", "."},
		"lint":      {"vet", "./..."},
		"typecheck": {"build", "-o", "/dev/null", "./..."},
		"tc":        {"build", "-o", "/dev/null", "./..."},
	}

	for _, variant := range GetCommandVariants(command) {
		if goCmd, ok := goCommands[variant]; ok {
			cmdArgs := append(goCmd, args...)
			cmd := exec.Command("go", cmdArgs...)
			cmd.Dir = g.dir
			return cmd
		}
	}

	return nil
}

// GradleSource for Gradle projects
type GradleSource struct {
	baseSource
}

func NewGradleSource(dir string) CommandSource {
	if !FileExists(filepath.Join(dir, "build.gradle")) &&
		!FileExists(filepath.Join(dir, "build.gradle.kts")) {
		return nil
	}

	return &GradleSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "Gradle",
			priority: 10,
		},
	}
}

func (g *GradleSource) ListCommands() map[string]CommandInfo {
	gradleExec := "gradle"
	if FileExists(filepath.Join(g.dir, "gradlew")) {
		gradleExec = "./gradlew"
	}
	return map[string]CommandInfo{
		"build":   {Description: "Build the project", Execution: gradleExec + " build"},
		"run":     {Description: "Run the project", Execution: gradleExec + " run"},
		"test":    {Description: "Run tests", Execution: gradleExec + " test"},
		"clean":   {Description: "Clean build artifacts", Execution: gradleExec + " clean"},
		"check":   {Description: "Run checks", Execution: gradleExec + " check"},
		"setup":   {Description: "Download dependencies", Execution: gradleExec + " build"},
		"install": {Description: "Install application (requires application plugin)", Execution: gradleExec + " installDist"},
	}
}

func (g *GradleSource) FindCommand(command string, args []string) *exec.Cmd {
	gradleExec := "gradle"
	if FileExists(filepath.Join(g.dir, "gradlew")) {
		gradleExec = "./gradlew"
	}

	gradleCommands := map[string]string{
		"build":   "build",
		"run":     "run",
		"test":    "test",
		"clean":   "clean",
		"check":   "check",
		"setup":   "build",
		"install": "installDist",
	}

	for _, variant := range GetCommandVariants(command) {
		if gradleCmd, ok := gradleCommands[variant]; ok {
			var cmdArgs []string
			// Handle commands with multiple parts (like "dependencies --write-locks")
			if strings.Contains(gradleCmd, " ") {
				parts := strings.Fields(gradleCmd)
				cmdArgs = append(parts, args...)
			} else {
				cmdArgs = append([]string{gradleCmd}, args...)
			}
			cmd := exec.Command(gradleExec, cmdArgs...)
			cmd.Dir = g.dir
			return cmd
		}
	}

	return nil
}

// MavenSource for Maven projects
type MavenSource struct {
	baseSource
}

func NewMavenSource(dir string) CommandSource {
	if !FileExists(filepath.Join(dir, "pom.xml")) {
		return nil
	}

	return &MavenSource{
		baseSource: baseSource{
			dir:      dir,
			name:     "Maven",
			priority: 10,
		},
	}
}

func (m *MavenSource) ListCommands() map[string]CommandInfo {
	mvnExec := "mvn"
	if FileExists(filepath.Join(m.dir, "mvnw")) {
		mvnExec = "./mvnw"
	}
	return map[string]CommandInfo{
		"build":   {Description: "Build the project", Execution: mvnExec + " compile"},
		"run":     {Description: "Run the project", Execution: mvnExec + " exec:java"},
		"test":    {Description: "Run tests", Execution: mvnExec + " test"},
		"clean":   {Description: "Clean build artifacts", Execution: mvnExec + " clean"},
		"setup":   {Description: "Download dependencies", Execution: mvnExec + " dependency:resolve"},
		"install": {Description: "Install to local Maven repository", Execution: mvnExec + " install"},
		"package": {Description: "Package the project", Execution: mvnExec + " package"},
	}
}

func (m *MavenSource) FindCommand(command string, args []string) *exec.Cmd {
	mvnExec := "mvn"
	if FileExists(filepath.Join(m.dir, "mvnw")) {
		mvnExec = "./mvnw"
	}

	mavenCommands := map[string]string{
		"build":   "compile",
		"run":     "exec:java",
		"test":    "test",
		"clean":   "clean",
		"setup":   "dependency:resolve",
		"install": "install",
		"package": "package",
	}

	for _, variant := range GetCommandVariants(command) {
		if mvnCmd, ok := mavenCommands[variant]; ok {
			cmdArgs := append([]string{mvnCmd}, args...)
			cmd := exec.Command(mvnExec, cmdArgs...)
			cmd.Dir = m.dir
			return cmd
		}
	}

	return nil
}
