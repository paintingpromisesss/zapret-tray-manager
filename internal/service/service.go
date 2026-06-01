package service

import (
	"log/slog"
	"strings"

	"zapret-tray-manager/internal/winexec"
)

type Service struct {
	logger *slog.Logger
	runner CommandRunner
	root   string
}

type Option func(*Service)

func WithRunner(runner CommandRunner) Option {
	return func(s *Service) {
		if runner != nil {
			s.runner = runner
		}
	}
}

func New(root string, logger *slog.Logger, opts ...Option) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Service{
		root:   root,
		logger: logger,
		runner: winexec.NewRunner(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) SetRoot(root string) {
	s.root = root
}

func (s *Service) run(name string, args ...string) error {
	_, err := s.runner.Run(name, args...)
	//nolint:wrapcheck // Errors wrapps higher up the call stack, so we don't need to add more context here.
	return err
}

func isBenignDeleteError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "1060") || strings.Contains(msg, "1072")
}

func isBenignStopError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "1062") || strings.Contains(msg, "2185")
}
