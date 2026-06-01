package app

import (
	"os"
	"path/filepath"
	"strings"
)

type UserLocalRoot struct {
	Path  string
	Valid bool
}

func (a *App) UserLocalRoots() []UserLocalRoot {
	a.mu.Lock()
	paths := make([]string, len(a.cfg.UserLocalRoots))
	copy(paths, a.cfg.UserLocalRoots)
	a.mu.Unlock()

	out := make([]UserLocalRoot, 0, len(paths))
	for _, p := range paths {
		_, statErr := os.Stat(p)
		out = append(out, UserLocalRoot{
			Path:  p,
			Valid: statErr == nil,
		})
	}
	return out
}

func (a *App) AddUserLocalRoot(root string) error {
	root = cleanPath(root)
	if root == "" {
		return nil
	}
	a.mu.Lock()
	for _, p := range a.cfg.UserLocalRoots {
		if strings.EqualFold(filepath.Clean(p), filepath.Clean(root)) {
			a.mu.Unlock()
			return nil
		}
	}
	a.cfg.UserLocalRoots = append(a.cfg.UserLocalRoots, root)
	a.mu.Unlock()
	return a.saveConfig()
}

func (a *App) RemoveUserLocalRoot(root string) error {
	root = cleanPath(root)
	a.mu.Lock()
	filtered := a.cfg.UserLocalRoots[:0]
	for _, p := range a.cfg.UserLocalRoots {
		if !strings.EqualFold(filepath.Clean(p), filepath.Clean(root)) {
			filtered = append(filtered, p)
		}
	}
	a.cfg.UserLocalRoots = filtered
	a.mu.Unlock()
	return a.saveConfig()
}

func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(p)
}
