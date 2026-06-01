//go:build windows

package winexec

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/text/encoding/charmap"
)

type Result struct {
	Name   string
	Output string
	Args   []string
}

type Runner struct{}

func NewRunner() *Runner {
	return &Runner{}
}

func Command(name string, args ...string) *exec.Cmd {
	//nolint:gosec,noctx // Central runner; callers pass fixed system commands. Use CommandContext when cancellation is needed.
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	//nolint:gosec // Central runner; callers pass fixed system commands (sc, net, ...).
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

func (r *Runner) Run(name string, args ...string) (Result, error) {
	return r.RunContext(context.Background(), name, args...)
}

func (r *Runner) RunContext(ctx context.Context, name string, args ...string) (Result, error) {
	result := Result{Name: name, Args: append([]string(nil), args...)}
	cmd := CommandContext(ctx, name, args...)
	raw, err := cmd.CombinedOutput()
	result.Output = strings.TrimSpace(decodeCP866(raw))
	if err == nil {
		return result, nil
	}
	if result.Output == "" {
		return result, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return result, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, result.Output)
}

func decodeCP866(b []byte) string {
	decoded, err := charmap.CodePage866.NewDecoder().Bytes(b)
	if err != nil {
		return string(b)
	}
	return string(decoded)
}
