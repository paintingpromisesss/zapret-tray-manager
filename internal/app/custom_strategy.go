package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"zapret-tray-manager/internal/config"
)

const customStrategySuffix = " (custom)"

func CustomStrategiesDir() string {
	return filepath.Join(config.ExecutableDir(), "custom_strategies")
}

func (a *App) AddCustomStrategy(srcPath string) error {
	dir := CustomStrategiesDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create custom strategies dir: %w", err)
	}

	base := filepath.Base(srcPath)
	ext := filepath.Ext(base)
	nameNoExt := strings.TrimSuffix(base, ext)

	destName := pickCustomName(dir, nameNoExt, ext)
	destPath := filepath.Join(dir, destName)

	if err := copyFileApp(srcPath, destPath); err != nil {
		return fmt.Errorf("copy strategy: %w", err)
	}

	a.mu.Lock()
	a.cfg.CustomStrategies = append(a.cfg.CustomStrategies, destName)
	a.mu.Unlock()

	a.logger.Info("custom strategy added", "name", destName)
	return a.saveConfig()
}

func pickCustomName(dir, nameNoExt, ext string) string {
	candidate := nameNoExt + customStrategySuffix + ext
	if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
		return candidate
	}
	for n := 2; ; n++ {
		candidate = fmt.Sprintf("%s (custom-%d)%s", nameNoExt, n, ext)
		if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
}

func (a *App) CustomStrategies() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, len(a.cfg.CustomStrategies))
	copy(out, a.cfg.CustomStrategies)
	return out
}

func copyFileApp(src, dst string) error {
	in, err := os.Open(src) //nolint:gosec // src/dst are app-controlled strategy file paths.
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) //nolint:gosec // see above
	if err != nil {
		return fmt.Errorf("failed to open destination file: %w", err)
	}

	if _, err = io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("failed to copy file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close destination file: %w", err)
	}
	return nil
}
