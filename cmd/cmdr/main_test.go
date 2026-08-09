package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestRunInformationalFlags(t *testing.T) {
	previousVersion := version
	version = "1.2.3"
	t.Cleanup(func() { version = previousVersion })

	tests := []struct {
		name       string
		args       []string
		wantStatus int
		wantOutput string
		wantError  string
	}{
		{name: "help", args: []string{"--help"}, wantError: "Usage: cmdr"},
		{name: "version", args: []string{"--version"}, wantOutput: "cmdr version 1.2.3"},
		{name: "unknown option", args: []string{"--unknown"}, wantStatus: 1, wantError: "Unknown option: --unknown"},
		{name: "list with command", args: []string{"--list", "test"}, wantStatus: 1, wantError: "--list flag must appear without a command"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := run(test.args, strings.NewReader(""), &stdout, &stderr)
			if status != test.wantStatus {
				t.Fatalf("run() status = %d, want %d", status, test.wantStatus)
			}
			if !strings.Contains(stdout.String(), test.wantOutput) {
				t.Errorf("stdout = %q, want it to contain %q", stdout.String(), test.wantOutput)
			}
			if !strings.Contains(stderr.String(), test.wantError) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), test.wantError)
			}
		})
	}
}

func TestRunListsCommandsThroughInjectedWriter(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout bytes.Buffer

	if status := run(nil, strings.NewReader(""), &stdout, io.Discard); status != 0 {
		t.Fatalf("run() status = %d, want 0", status)
	}
	if !strings.Contains(stdout.String(), "Available commands for this project:") {
		t.Fatalf("stdout = %q, want command list", stdout.String())
	}
}

func TestRunReturnsInteractiveError(t *testing.T) {
	previousRunInteractive := runInteractive
	runInteractive = func() error { return errors.New("terminal unavailable") }
	t.Cleanup(func() { runInteractive = previousRunInteractive })

	var stderr bytes.Buffer
	if status := run([]string{"--interactive"}, strings.NewReader(""), io.Discard, &stderr); status != 1 {
		t.Fatalf("run() status = %d, want 1", status)
	}
	if !strings.Contains(stderr.String(), "terminal unavailable") {
		t.Fatalf("stderr = %q, want interactive error", stderr.String())
	}
}
