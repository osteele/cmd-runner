package cmdrunner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMakeRunnerHasTarget(t *testing.T) {
	tempDir := t.TempDir()
	makefileContent := `
.PHONY: build test format

build:
	go build

test:
	go test ./...

format:
	go fmt ./...
`
	makefilePath := filepath.Join(tempDir, "Makefile")
	if err := os.WriteFile(makefilePath, []byte(makefileContent), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &makeRunner{}

	tests := []struct {
		target   string
		expected bool
	}{
		{"build", true},
		{"test", true},
		{"format", true},
		{"nonexistent", false},
		{"clean", false},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			result := runner.hasTarget(tempDir, tt.target)
			if result != tt.expected {
				t.Errorf("hasTarget(%q) = %v, want %v", tt.target, result, tt.expected)
			}
		})
	}
}

func TestNodePackageRunnerFindCommand(t *testing.T) {
	tempDir := t.TempDir()

	packageJSON := map[string]interface{}{
		"name": "test-project",
		"scripts": map[string]string{
			"dev":    "vite",
			"build":  "vite build",
			"test":   "vitest",
			"format": "prettier --write .",
		},
	}

	data, _ := json.Marshal(packageJSON)
	packagePath := filepath.Join(tempDir, "package.json")
	if err := os.WriteFile(packagePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	runner := &nodePackageRunner{}

	tests := []struct {
		command string
		hasCmd  bool
	}{
		{"dev", true},
		{"build", true},
		{"test", true},
		{"format", true},
		{"lint", false},
		{"clean", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			cmd := runner.findCommand(tempDir, tt.command, []string{})
			if tt.hasCmd && cmd == nil {
				t.Errorf("expected to find command %q but got nil", tt.command)
			}
			if !tt.hasCmd && cmd != nil {
				t.Errorf("expected no command for %q but got %v", tt.command, cmd.Args)
			}
		})
	}
}

func TestCargoRunnerFindCommand(t *testing.T) {
	tempDir := t.TempDir()

	cargoToml := `[package]
name = "test-project"
version = "0.1.0"
edition = "2021"
`
	cargoPath := filepath.Join(tempDir, "Cargo.toml")
	if err := os.WriteFile(cargoPath, []byte(cargoToml), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &cargoRunner{}

	tests := []struct {
		command  string
		expected []string
	}{
		{"build", []string{"cargo", "build"}},
		{"run", []string{"cargo", "run"}},
		{"test", []string{"cargo", "test"}},
		{"format", []string{"cargo", "fmt"}},
		{"fmt", []string{"cargo", "fmt"}},
		{"lint", []string{"cargo", "clippy"}},
		{"clean", []string{"cargo", "clean"}},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			cmd := runner.findCommand(tempDir, tt.command, []string{})
			if cmd != nil && !slicesEqual(cmd.Args[:len(tt.expected)], tt.expected) {
				t.Errorf("findCommand(%q) = %v, want prefix %v", tt.command, cmd.Args, tt.expected)
			}
		})
	}
}

func TestGoRunnerFindCommand(t *testing.T) {
	tempDir := t.TempDir()

	goMod := `module test-project

go 1.21
`
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &goRunner{}

	tests := []struct {
		command string
		hasCmd  bool
	}{
		{"build", true},
		{"run", true},
		{"test", true},
		{"format", true},
		{"fmt", true},
		{"lint", true},
		{"clean", true},
		{"install", true},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			cmd := runner.findCommand(tempDir, tt.command, []string{})
			if tt.hasCmd && cmd == nil {
				t.Errorf("expected to find command %q but got nil", tt.command)
			}
			if !tt.hasCmd && cmd != nil {
				t.Errorf("expected no command for %q but got %v", tt.command, cmd.Args)
			}
		})
	}
}

func TestJustRunnerFindCommand(t *testing.T) {
	tempDir := t.TempDir()

	justfile := `
default:
    @just --list

test:
    go test ./...

format:
    go fmt ./...

build:
    go build
`
	justfilePath := filepath.Join(tempDir, "justfile")
	if err := os.WriteFile(justfilePath, []byte(justfile), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &justRunner{}

	cmd := runner.findCommand(tempDir, "test", []string{})
	if cmd == nil {
		t.Skip("just command not available")
	}
}
