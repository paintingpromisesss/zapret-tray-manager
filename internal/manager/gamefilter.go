package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type GameFilterMode string

var (
	ErrUnknownGameFilterMode = fmt.Errorf("unknown game filter mode")
)

const (
	GameFilterDisabled GameFilterMode = "disabled"
	GameFilterAll      GameFilterMode = "all"
	GameFilterTCP      GameFilterMode = "tcp"
	GameFilterUDP      GameFilterMode = "udp"
)

type GameFilterPorts struct {
	All string
	TCP string
	UDP string
}

func (m GameFilterMode) Ports() GameFilterPorts {
	switch m {
	case GameFilterAll:
		return GameFilterPorts{
			All: "1024-65535",
			TCP: "1024-65535",
			UDP: "1024-65535",
		}
	case GameFilterTCP:
		return GameFilterPorts{
			All: "1024-65535",
			TCP: "1024-65535",
			UDP: "12",
		}
	case GameFilterUDP:
		return GameFilterPorts{
			All: "1024-65535",
			TCP: "12",
			UDP: "1024-65535",
		}
	default:
		return GameFilterPorts{
			All: "12",
			TCP: "12",
			UDP: "12",
		}
	}
}

func (m *Manager) GameFilterMode() GameFilterMode {
	content, err := os.ReadFile(m.gameFilterModePath())
	if err != nil {
		return GameFilterDisabled
	}

	switch strings.ToLower(strings.TrimSpace(string(content))) {
	case "all":
		return GameFilterAll
	case "tcp":
		return GameFilterTCP
	default:
		return GameFilterUDP
	}
}

func (m *Manager) SetGameFilterMode(mode GameFilterMode) error {
	path := m.gameFilterModePath()

	switch mode {
	case GameFilterDisabled:
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to remove game filter mode file: %w", err)
		}
	case GameFilterAll:
		if err := os.WriteFile(path, []byte("all\r\n"), 0600); err != nil {
			return fmt.Errorf("failed to write game filter mode file: %w", err)
		}
	case GameFilterTCP:
		if err := os.WriteFile(path, []byte("tcp\r\n"), 0600); err != nil {
			return fmt.Errorf("failed to write game filter mode file: %w", err)
		}
	case GameFilterUDP:
		if err := os.WriteFile(path, []byte("udp\r\n"), 0600); err != nil {
			return fmt.Errorf("failed to write game filter mode file: %w", err)
		}
	default:
		return fmt.Errorf("%w: %s", ErrUnknownGameFilterMode, mode)
	}
	return nil
}

func (m *Manager) gameFilterModePath() string {
	return filepath.Join(m.Root, "utils", "game_filter.enabled")
}
