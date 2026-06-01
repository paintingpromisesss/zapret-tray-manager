package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"zapret-tray-manager/internal/manager"
	"zapret-tray-manager/internal/service"
	"zapret-tray-manager/internal/strategy"
)

var (
	ErrUnknownRunningStrategy = errors.New("running strategy is unknown")
)

func (a *App) InstallStrategy(strategyName string) error {
	s, err := a.findStrategy(strategyName)
	if err != nil {
		return err
	}
	cfg := a.Config()
	return a.manager.Install(s, cfg.ZapretAutoRunEnabled)
}

func (a *App) ToggleRunStop() error {
	status, err := a.manager.ServiceStatus()
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}
	if status.Zapret == service.StateRunning || status.Zapret == service.StateStartPending {
		if err := a.manager.Stop(); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
		return nil
	}
	if err := a.manager.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	return nil
}

func (a *App) Remove() error {
	return a.manager.Remove()
}

func (a *App) SetGameFilter(mode manager.GameFilterMode) error {
	status, err := a.manager.ServiceStatus()
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}
	if err := a.manager.SetGameFilterMode(mode); err != nil {
		return fmt.Errorf("failed to set game filter mode: %w", err)
	}
	a.mu.Lock()
	globalEnabled := a.cfg.GlobalSettingsEnabled
	if globalEnabled {
		a.cfg.GlobalGameFilter = string(mode)
	}
	a.mu.Unlock()
	if globalEnabled {
		if err := a.saveConfig(); err != nil {
			return fmt.Errorf("failed to save global game filter: %w", err)
		}
	}
	return a.reconfigureIfRunning(status)
}

func (a *App) SetIPSet(mode manager.IPSetMode) error {
	status, err := a.manager.ServiceStatus()
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}
	if mode == manager.IPSetLoaded {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := a.manager.EnsureIPSetLoaded(ctx); err != nil {
			return fmt.Errorf("failed to set IP set mode: %w", err)
		}
	} else {
		if err := a.manager.SetIPSetMode(mode); err != nil {
			return fmt.Errorf("failed to set IP set mode: %w", err)
		}
	}
	a.mu.Lock()
	globalEnabled := a.cfg.GlobalSettingsEnabled
	if globalEnabled {
		a.cfg.GlobalIPSetMode = string(mode)
	}
	a.mu.Unlock()
	if globalEnabled {
		if err := a.saveConfig(); err != nil {
			return fmt.Errorf("failed to save global ipset mode: %w", err)
		}
	}
	return a.reconfigureIfRunning(status)
}

func (a *App) reconfigureIfRunning(status service.Status) error {
	if status.Zapret != service.StateRunning {
		return nil
	}

	strategyName := status.InstalledStrategy
	if strategyName == "" {
		return ErrUnknownRunningStrategy
	}

	s, err := a.findStrategy(strategyName)
	if err != nil {
		return err
	}

	cfg := a.Config()
	return a.manager.ConfigureStrategy(s, cfg.ZapretAutoRunEnabled, true)
}

func (a *App) reconfigureAfterRootSwitch(status service.Status) error {
	if status.Zapret == service.StateNotInstalled || status.InstalledStrategy == "" {
		return nil
	}

	s, err := a.findStrategy(status.InstalledStrategy)
	if err != nil {
		a.logger.Warn("installed strategy is missing in new zapret root; stopping service", "strategy", status.InstalledStrategy, "error", err)
		stopErr := a.manager.Stop()
		if stopErr != nil {
			return fmt.Errorf("failed to find installed strategy (stop failed: %w)", stopErr)
		}
		return fmt.Errorf("failed to find installed strategy: %w", err)
	}

	start := status.Zapret == service.StateRunning || status.Zapret == service.StateStartPending
	cfg := a.Config()
	return a.manager.ConfigureStrategy(s, cfg.ZapretAutoRunEnabled, start)
}

func (a *App) findStrategy(name string) (strategy.Strategy, error) {
	name = strings.TrimSpace(name)
	strategies, err := strategy.List(a.manager.Root, a.CustomStrategies(), CustomStrategiesDir())
	if err != nil {
		return strategy.Strategy{}, fmt.Errorf("failed to list strategies: %w", err)
	}

	for _, s := range strategies {
		base := strings.TrimSuffix(s.Name, filepath.Ext(s.Name))
		if strings.EqualFold(s.Name, name) || strings.EqualFold(base, name) {
			return s, nil
		}
	}
	return strategy.Strategy{}, fmt.Errorf("%w: %s", ErrStrategyNotFound, name)
}
