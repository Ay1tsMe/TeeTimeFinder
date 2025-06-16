package cmd

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// setupLogging turns on Bubble Tea’s file logger when the –v/--verbose flag
// is present.  It returns the opened *os.File so main() can defer Close().
func setupLogging() (*os.File, error) {
	if !verboseMode { // flag not set keep stdout clean
		return nil, nil
	}

	// ~/.config/TeeTimeFinder/debug.log  (mkdir -p if necessary)
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	logDir := filepath.Join(cfgDir, "TeeTimeFinder")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, err
	}
	logPath := filepath.Join(logDir, "debug.log")

	// Write “TeeTimeFinder ” in front of every line
	return tea.LogToFile(logPath, "TeeTimeFinder")
}
