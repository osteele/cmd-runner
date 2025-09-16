package internal

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

// TerminalManager handles terminal mode switching and input
type TerminalManager struct {
	oldState *term.State
	fd       int
}

// NewTerminalManager creates a new terminal manager
func NewTerminalManager() *TerminalManager {
	return &TerminalManager{
		fd: int(os.Stdin.Fd()),
	}
}

// SetRawMode puts the terminal in raw mode for single-key input
func (tm *TerminalManager) SetRawMode() error {
	var err error
	tm.oldState, err = term.MakeRaw(tm.fd)
	return err
}

// RestoreMode restores the terminal to its original mode
func (tm *TerminalManager) RestoreMode() error {
	if tm.oldState != nil {
		return term.Restore(tm.fd, tm.oldState)
	}
	return nil
}

// ReadKey reads a single key from the terminal
func (tm *TerminalManager) ReadKey() (rune, error) {
	b := make([]byte, 1)
	_, err := os.Stdin.Read(b)
	if err != nil {
		return 0, err
	}

	// Handle special keys
	if b[0] == 3 { // Ctrl+C
		return 0, fmt.Errorf("interrupt")
	}
	if b[0] == 27 { // ESC
		return 0, fmt.Errorf("escape")
	}

	return rune(b[0]), nil
}

// SetupSignalHandling sets up signal handlers for clean exit
func (tm *TerminalManager) SetupSignalHandling(cleanup func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cleanup()
		os.Exit(0)
	}()
}

// ClearScreen clears the terminal screen
func ClearScreen() {
	fmt.Print("\033[2J\033[H")
}

// MoveCursorUp moves the cursor up n lines
func MoveCursorUp(n int) {
	fmt.Printf("\033[%dA", n)
}

// ClearLine clears the current line
func ClearLine() {
	fmt.Print("\033[2K\r")
}
