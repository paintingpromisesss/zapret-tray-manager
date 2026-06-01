package logging

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const appName = "zapret-tray-manager"

type Logs struct {
	Logger *slog.Logger
	file   *os.File
	Path   string
}

func Setup() (*Logs, error) {
	path := DefaultPath(time.Now())

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	//nolint:gosec // log path is generated internally by DefaultPath() func and doesn't include user input
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	slog.SetDefault(logger)

	logger.Info("zapret-tray-manager started", "log_path", path)

	return &Logs{
		Logger: logger,
		Path:   path,
		file:   file,
	}, nil
}

func (l *Logs) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("close log file: %w", err)
	}
	return nil
}

func DefaultDir() string {
	if executable, err := os.Executable(); err == nil && executable != "" {
		return filepath.Join(filepath.Dir(executable), "logs")
	}

	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		return filepath.Join(cwd, "logs")
	}

	return "logs"
}

func DefaultFileName(now time.Time) string {
	return appName + "-" + now.Format("2006-01-02_15-04-05") + ".log"
}

func DefaultPath(now time.Time) string {
	return filepath.Join(DefaultDir(), DefaultFileName(now))
}
