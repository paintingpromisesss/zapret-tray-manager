package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"zapret-tray-manager/internal/strategy"
)

var (
	ErrStrategyNameEmpty = errors.New("strategy name is empty")
)

type GameFilterPorts struct {
	All string
	TCP string
	UDP string
}

func (s *Service) Install(st strategy.Strategy, ports GameFilterPorts, autoRun bool) error {
	return s.Configure(st, ports, autoRun, true)
}

func (s *Service) Configure(st strategy.Strategy, ports GameFilterPorts, autoRun bool, start bool) error {
	parsed, err := strategy.Parse(st, s.root, strategy.GameFilterPorts{
		All: ports.All,
		TCP: ports.TCP,
		UDP: ports.UDP,
	})
	if err != nil {
		return fmt.Errorf("parse strategy: %w", err)
	}

	if err := s.enableTCPTimestamps(); err != nil {
		return fmt.Errorf("enable TCP timestamps: %w", err)
	}

	if err := s.ensureUserFiles(); err != nil {
		return fmt.Errorf("ensure user files: %w", err)
	}

	binPath := buildServiceBinPath(filepath.Join(s.root, "bin", "winws.exe"), parsed.Args)
	if err := s.installOrUpdate(binPath, autoRun); err != nil {
		return fmt.Errorf("install/update zapret service: %w", err)
	}

	if err := s.run("sc.exe", "description", "zapret", "Zapret DPI bypass software"); err != nil {
		return fmt.Errorf("set zapret description: %w", err)
	}

	if err := s.run("sc.exe", "failure", "zapret", "reset=86400", "actions=restart/5000"); err != nil {
		s.logger.Warn("set zapret failure actions failed (non-fatal)", "error", err)
	}

	if start {
		if err := s.run("sc.exe", "start", "zapret"); err != nil {
			return fmt.Errorf("start zapret: %w", err)
		}
	} else {
		s.logger.Info("zapret service configured without start", "strategy", st.Name)
	}

	strategyName := strings.TrimSuffix(st.Name, filepath.Ext(st.Name))
	if err := s.SetInstalledStrategyName(strategyName); err != nil {
		return fmt.Errorf("save installed strategy: %w", err)
	}
	return nil
}

func (s *Service) SetInstalledStrategyName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrStrategyNameEmpty
	}
	return s.run(
		"reg.exe", "add",
		`HKLM\System\CurrentControlSet\Services\zapret`,
		"/v", "zapret-discord-youtube",
		"/t", "REG_SZ",
		"/d", name,
		"/f",
	)
}

func (s *Service) SetAutoRun(autoRun bool) error {
	state, err := s.queryState("zapret")
	if err != nil {
		return err
	}
	if state == StateNotInstalled {
		return ErrZapretServiceNotInstalled
	}
	return s.run("sc.exe", "config", "zapret", "start="+startMode(autoRun))
}

func (s *Service) installOrUpdate(binPath string, autoRun bool) error {
	state, err := s.queryState("zapret")
	if err != nil {
		return err
	}

	if state == StateNotInstalled {
		return s.run(
			"sc.exe", "create", "zapret",
			"binPath="+binPath,
			"DisplayName=zapret",
			"start="+startMode(autoRun),
		)
	}

	if state == StateRunning || state == StateStartPending || state == StateStopPending {
		if err := s.run("net.exe", "stop", "zapret"); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
	}

	return s.run(
		"sc.exe", "config", "zapret",
		"binPath="+binPath,
		"DisplayName=zapret",
		"start="+startMode(autoRun),
	)
}

func (s *Service) ensureUserFiles() error {
	listsPath := filepath.Join(s.root, "lists")
	files := map[string]string{
		"ipset-exclude-user.txt": "203.0.113.113/32\r\n",
		"list-general-user.txt":  "domain.example.abc\r\n",
		"list-exclude-user.txt":  "domain.example.abc\r\n",
	}
	for name, placeholder := range files {
		path := filepath.Join(listsPath, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(placeholder), 0600); err != nil {
				return fmt.Errorf("create %s: %w", name, err)
			}
		}
	}
	return nil
}

func (s *Service) enableTCPTimestamps() error {
	return s.run("netsh.exe", "interface", "tcp", "set", "global", "timestamps=enabled")
}

func startMode(autoRun bool) string {
	if autoRun {
		return "auto"
	}
	return "demand"
}
