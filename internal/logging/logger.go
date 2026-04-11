// Package logging provides a file-based debug logger for the Obsidian CLI.
// It wraps log/slog and is a no-op when disabled, so call sites can sprinkle
// logging.Debug/Info/Error everywhere without worrying about runtime cost.
//
// The logger is a singleton initialized once in tui.Run() from config.
// It writes to a file only — never stdout — so it is safe to call from
// inside the Bubble Tea TUI without corrupting the terminal.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	logger *slog.Logger
	file   *os.File
	mu     sync.Mutex
)

// Init sets up the singleton logger. Safe to call multiple times — subsequent
// calls close the previous file and re-initialize.
//
// If enabled is false, all subsequent Debug/Info/Warn/Error calls become no-ops
// (cost: one boolean check).
//
// The logFile path supports ~ expansion. Parent directories are created if
// missing. If truncateOnStart is true, the file is wiped on each Init.
func Init(enabled bool, logFile, level string, truncateOnStart bool) error {
	mu.Lock()
	defer mu.Unlock()

	// Close any existing file
	if file != nil {
		file.Close()
		file = nil
	}
	logger = nil

	if !enabled {
		return nil
	}

	// Expand ~ in path
	if strings.HasPrefix(logFile, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			logFile = filepath.Join(home, logFile[2:])
		}
	}

	// Ensure parent directory exists
	if dir := filepath.Dir(logFile); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	// Open the log file
	flags := os.O_CREATE | os.O_WRONLY
	if truncateOnStart {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_APPEND
	}

	f, err := os.OpenFile(logFile, flags, 0o644)
	if err != nil {
		return err
	}
	file = f

	// Parse level
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: lvl,
	})
	logger = slog.New(handler)
	return nil
}

// Close flushes and closes the log file. Call at app exit.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		file.Sync()
		file.Close()
		file = nil
	}
	logger = nil
}

// Enabled returns true if the logger is active.
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return logger != nil
}

// LogFile returns the path of the currently open log file, or empty string
// if logging is disabled.
func LogFile() string {
	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return ""
	}
	return file.Name()
}

// Debug logs at debug level. No-op if logger is disabled.
func Debug(msg string, args ...any) {
	if logger != nil {
		logger.Debug(msg, args...)
	}
}

// Info logs at info level. No-op if logger is disabled.
func Info(msg string, args ...any) {
	if logger != nil {
		logger.Info(msg, args...)
	}
}

// Warn logs at warn level. No-op if logger is disabled.
func Warn(msg string, args ...any) {
	if logger != nil {
		logger.Warn(msg, args...)
	}
}

// Error logs at error level. No-op if logger is disabled.
func Error(msg string, args ...any) {
	if logger != nil {
		logger.Error(msg, args...)
	}
}

// Writer returns the underlying log file writer, or io.Discard if disabled.
// Useful for redirecting subprocess output or third-party loggers.
func Writer() io.Writer {
	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return io.Discard
	}
	return file
}
