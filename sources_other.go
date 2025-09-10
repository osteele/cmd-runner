package cmdrunner

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

func (c *CargoSource) ListCommands() map[string]string {
	return map[string]string{
		"build":  "cargo build",
		"run":    "cargo run",
		"test":   "cargo test",
		"check":  "cargo check",
		"format": "cargo fmt",
		"lint":   "cargo clippy",
		"clean":  "cargo clean",
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
		"install":   "install",
		"publish":   "publish",
	}
	
	for _, variant := range GetCommandVariants(command) {
		if cargoCmd, ok := cargoCommands[variant]; ok {
			cmdArgs := append([]string{cargoCmd}, args...)
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

func (g *GoSource) ListCommands() map[string]string {
	return map[string]string{
		"build":  "go build",
		"run":    "go run .",
		"test":   "go test ./...",
		"format": "go fmt ./...",
		"lint":   "go vet ./...",
		"clean":  "go clean",
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
		"install":   {"mod", "download"},
		"lint":      {"vet", "./..."},
		"typecheck": {"build", "-o", "/dev/null"},
		"tc":        {"build", "-o", "/dev/null"},
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

func (g *GradleSource) ListCommands() map[string]string {
	return map[string]string{
		"build": "gradle build",
		"run":   "gradle run",
		"test":  "gradle test",
		"clean": "gradle clean",
		"check": "gradle check",
	}
}

func (g *GradleSource) FindCommand(command string, args []string) *exec.Cmd {
	gradleExec := "gradle"
	if FileExists(filepath.Join(g.dir, "gradlew")) {
		gradleExec = "./gradlew"
	}
	
	gradleCommands := map[string]string{
		"build": "build",
		"run":   "run",
		"test":  "test",
		"clean": "clean",
		"check": "check",
	}
	
	for _, variant := range GetCommandVariants(command) {
		if gradleCmd, ok := gradleCommands[variant]; ok {
			cmdArgs := append([]string{gradleCmd}, args...)
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

func (m *MavenSource) ListCommands() map[string]string {
	return map[string]string{
		"build":   "mvn compile",
		"run":     "mvn exec:java",
		"test":    "mvn test",
		"clean":   "mvn clean",
		"install": "mvn install",
		"package": "mvn package",
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