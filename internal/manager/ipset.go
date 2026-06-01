package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrUnknownIPSetMode = fmt.Errorf("unknown ipset mode")
)

type IPSetMode string

const (
	IPSetLoaded IPSetMode = "loaded"
	IPSetNone   IPSetMode = "none"
	IPSetAny    IPSetMode = "any"
)

func (m *Manager) IPSetMode() (IPSetMode, error) {
	content, err := os.ReadFile(m.ipsetPath())
	if err != nil {
		return "", fmt.Errorf("failed to read ipset file: %w", err)
	}

	switch strings.TrimSpace(string(content)) {
	case "":
		return IPSetAny, nil
	case ipsetNoneSentinel:
		return IPSetNone, nil
	default:
		return IPSetLoaded, nil
	}
}

func (m *Manager) SetIPSetMode(mode IPSetMode) error {
	currentMode, err := m.IPSetMode()
	if err != nil {
		return fmt.Errorf("failed to get current ipset mode: %w", err)
	}

	if currentMode == mode {
		return nil
	}

	switch mode {
	case IPSetLoaded:
		return m.setIPSetLoaded()
	case IPSetNone:
		return m.setIPsetNone(currentMode)
	case IPSetAny:
		return m.setIPSetAny(currentMode)
	default:
		return ErrUnknownIPSetMode
	}
}

func (m *Manager) setIPSetLoaded() error {
	backupPath := m.ipsetBackupPath()
	if _, err := os.Stat(backupPath); err == nil {
		return m.restoreIPSetBackup()
	}
	return nil
}

func (m *Manager) EnsureIPSetLoaded(ctx context.Context) error {
	backupPath := m.ipsetBackupPath()
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		if err := m.UpdateIPSet(ctx); err != nil {
			return err
		}
	}
	return m.SetIPSetMode(IPSetLoaded)
}

func (m *Manager) setIPsetNone(currentMode IPSetMode) error {
	if currentMode == IPSetLoaded {
		if err := m.backupIPSet(); err != nil {
			return fmt.Errorf("failed to backup ipset-all.txt: %w", err)
		}
	}
	if err := os.WriteFile(m.ipsetPath(), []byte(ipsetNoneSentinel+"\r\n"), 0600); err != nil {
		return fmt.Errorf("failed to write ipset file: %w", err)
	}
	return nil
}

func (m *Manager) setIPSetAny(currentMode IPSetMode) error {
	if currentMode == IPSetLoaded {
		if err := m.backupIPSet(); err != nil {
			return fmt.Errorf("failed to backup ipset-all.txt: %w", err)
		}
	}
	if err := os.WriteFile(m.ipsetPath(), nil, 0600); err != nil {
		return fmt.Errorf("failed to write ipset file: %w", err)
	}
	return nil
}

func (m *Manager) UpdateIPSet(parentCtx context.Context) error {
	currentMode, err := m.IPSetMode()
	if err != nil {
		return fmt.Errorf("failed to get current ipset mode: %w", err)
	}

	downloadCtx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	var ipsetDownloadPath string
	if currentMode == IPSetLoaded {
		ipsetDownloadPath = m.ipsetPath()
	} else {
		ipsetDownloadPath = m.ipsetBackupPath()
	}

	_, err = m.httpClient.Download(downloadCtx, ipsetUpdateURL, ipsetDownloadPath)
	if err != nil {
		return fmt.Errorf("failed to download ipset: %w", err)
	}
	return nil
}

func (m *Manager) ipsetPath() string {
	return filepath.Join(m.Root, "lists", "ipset-all.txt")
}

func (m *Manager) backupIPSet() error {
	return copyFile(m.ipsetPath(), m.ipsetBackupPath())
}

func (m *Manager) restoreIPSetBackup() error {
	backupPath := m.ipsetBackupPath()
	if err := validatePath(backupPath, false); err != nil {
		return fmt.Errorf("failed to validate ipset backup path: %w", err)
	}
	return copyFile(backupPath, m.ipsetPath())
}

func (m *Manager) ipsetBackupPath() string {
	return m.ipsetPath() + ".backup"
}
