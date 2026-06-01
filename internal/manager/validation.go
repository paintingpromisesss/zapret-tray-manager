package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"zapret-tray-manager/internal/strategy"
)

var (
	ErrPathNotFound = errors.New("path not found")
	ErrNotADir      = errors.New("not a directory")
	ErrNotAFile     = errors.New("not a file")
)

func (m *Manager) Validate() error {
	if strings.TrimSpace(m.Root) == "" || m.Root == "." {
		return ErrRootEmpty
	}

	checks := []struct {
		path string
		name string
		dir  bool
	}{
		{path: m.Root, dir: true, name: "zapret root"},
		{path: filepath.Join(m.Root, "service.bat"), name: "service.bat"},
		{path: filepath.Join(m.Root, "bin"), dir: true, name: "bin dir"},
		{path: filepath.Join(m.Root, "bin", "winws.exe"), name: "bin/winws.exe"},
		{path: filepath.Join(m.Root, "lists"), dir: true, name: "lists dir"},
		{path: filepath.Join(m.Root, "lists", "ipset-all.txt"), name: "lists/ipset-all.txt"},
		{path: filepath.Join(m.Root, "utils"), dir: true, name: "utils dir"},
	}

	var validationErrors []error
	for _, check := range checks {
		if err := validatePath(check.path, check.dir); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("%s: %w", check.name, err))
		}
	}

	strategies, err := strategy.List(m.Root, nil, "")
	if err != nil {
		validationErrors = append(validationErrors, fmt.Errorf("strategies error: %w", err))
	} else if len(strategies) == 0 {
		validationErrors = append(validationErrors, strategy.ErrNotFound)
	}

	return errors.Join(validationErrors...)
}

func validatePath(path string, wantDir bool) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrPathNotFound
		}
		return fmt.Errorf("failed to stat %s: %w", path, err)
	}

	if wantDir && !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrNotADir, path)
	}
	if !wantDir && info.IsDir() {
		return fmt.Errorf("%w: %s", ErrNotAFile, path)
	}

	return nil
}
