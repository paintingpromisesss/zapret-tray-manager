//go:build !windows

package winexec

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Result struct {
	Name   string
	Args   []string
	Output string
}

type Runner struct{}

func NewRunner() *Runner {
	return &Runner{}
}

func Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func (r *Runner) Run(name string, args ...string) (Result, error) {
	return r.RunContext(context.Background(), name, args...)
}

func (r *Runner) RunContext(ctx context.Context, name string, args ...string) (Result, error) {
	result := Result{Name: name, Args: append([]string(nil), args...)}
	cmd := CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	result.Output = strings.TrimSpace(string(output))
	if err == nil {
		return result, nil
	}
	if result.Output == "" {
		return result, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return result, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, result.Output)
}
