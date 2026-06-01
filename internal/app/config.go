package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zapret-tray-manager/internal/manager"
	"zapret-tray-manager/internal/winexec"
)

func (a *App) SetRoot(root string) error {
	root = strings.TrimSpace(root)
	if root != "" {
		if abs, err := filepath.Abs(root); err == nil {
			root = filepath.Clean(abs)
		} else {
			root = filepath.Clean(root)
		}
	}

	oldRoot := a.manager.Root
	a.manager.SetRoot(root)
	if root != "" {
		_ = os.MkdirAll(filepath.Join(root, "utils"), 0700) //nolint:errcheck // see comment
	}
	if err := a.manager.Validate(); err != nil {
		a.manager.SetRoot(oldRoot)
		return fmt.Errorf("invalid zapret root: %w", err)
	}

	a.mu.Lock()
	a.cfg.CurrentRoot = root
	a.mu.Unlock()
	if err := a.saveConfig(); err != nil {
		return err
	}
	return a.applyGlobalSettingsIfEnabled()
}

func (a *App) applyGlobalSettingsIfEnabled() error {
	cfg := a.Config()
	if !cfg.GlobalSettingsEnabled {
		return nil
	}
	if cfg.GlobalGameFilter != "" {
		mode := manager.GameFilterMode(cfg.GlobalGameFilter)
		if err := a.manager.SetGameFilterMode(mode); err != nil {
			return fmt.Errorf("apply global game filter: %w", err)
		}
	}
	if cfg.GlobalIPSetMode == string(manager.IPSetLoaded) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := a.manager.EnsureIPSetLoaded(ctx); err != nil {
			return fmt.Errorf("apply global ipset: %w", err)
		}
	} else if cfg.GlobalIPSetMode != "" {
		mode := manager.IPSetMode(cfg.GlobalIPSetMode)
		if err := a.manager.SetIPSetMode(mode); err != nil {
			return fmt.Errorf("apply global ipset: %w", err)
		}
	}
	return nil
}

func (a *App) SyncAutoRunFromService() {
	actual, err := a.manager.QueryAutoRunService()
	if err != nil {
		a.logger.Warn("could not query service auto-run state", "error", err)
		return
	}
	a.mu.Lock()
	if a.cfg.ZapretAutoRunEnabled == actual {
		a.mu.Unlock()
		return
	}
	a.cfg.ZapretAutoRunEnabled = actual
	a.mu.Unlock()
	if err := a.saveConfig(); err != nil {
		a.logger.Warn("could not persist synced auto-run state", "error", err)
	}
}

func (a *App) SetAutoRunService(enabled bool) error {
	a.mu.Lock()
	a.cfg.ZapretAutoRunEnabled = enabled
	a.mu.Unlock()
	if err := a.saveConfig(); err != nil {
		return err
	}
	if err := a.manager.SetAutoRunService(enabled); err != nil {
		return fmt.Errorf("failed to set auto-run service: %w", err)
	}
	return nil
}

func (a *App) UpdateIPSet(ctx context.Context) error {
	if err := a.manager.UpdateIPSet(ctx); err != nil {
		return fmt.Errorf("failed to update IP set: %w", err)
	}
	return nil
}

func (a *App) SetVPNAfterAction(fn func()) {
	a.mu.Lock()
	a.vpnAfterAction = fn
	a.mu.Unlock()
}

func (a *App) SetVPNStopOnConnect(enabled bool) error {
	a.mu.Lock()
	a.cfg.VPNStopOnConnect = enabled
	a.mu.Unlock()
	if err := a.saveConfig(); err != nil {
		return err
	}
	a.syncTunWatcher()
	return nil
}

func (a *App) SetVPNStartOnDisconnect(enabled bool) error {
	a.mu.Lock()
	a.cfg.VPNStartOnDisconnect = enabled
	a.mu.Unlock()
	if err := a.saveConfig(); err != nil {
		return err
	}
	a.syncTunWatcher()
	return nil
}

func (a *App) syncTunWatcher() {
	cfg := a.Config()
	needWatch := cfg.VPNStopOnConnect || cfg.VPNStartOnDisconnect

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.tunWatcher == nil {
		a.tunWatcher = winexec.NewTunWatcher(a.onTunConnect, a.onTunDisconnect)
	}

	if needWatch {
		a.tunWatcher.Start()
	} else {
		a.tunWatcher.Stop()
	}
}

func (a *App) onTunConnect() {
	if !a.Config().VPNStopOnConnect {
		return
	}
	if err := a.RunExclusive("vpn: stop on tun connect", a.manager.Stop); err != nil {
		a.logger.Warn("vpn: failed to stop zapret on tun connect", "error", err)
	}
	a.runVPNAfterAction()
}

func (a *App) onTunDisconnect() {
	if !a.Config().VPNStartOnDisconnect {
		return
	}
	if err := a.RunExclusive("vpn: start on tun disconnect", a.manager.Start); err != nil {
		a.logger.Warn("vpn: failed to start zapret on tun disconnect", "error", err)
	}
	a.runVPNAfterAction()
}

func (a *App) runVPNAfterAction() {
	a.mu.Lock()
	fn := a.vpnAfterAction
	a.mu.Unlock()
	if fn != nil {
		fn()
	}
}

func (a *App) SetLanguage(lang string) error {
	a.mu.Lock()
	a.cfg.Language = lang
	a.mu.Unlock()
	return a.saveConfig()
}

func (a *App) SetGlobalSettingsEnabled(enabled bool) error {
	a.mu.Lock()
	a.cfg.GlobalSettingsEnabled = enabled
	if enabled {
		a.cfg.GlobalGameFilter = string(a.manager.GameFilterMode())
	}
	a.mu.Unlock()
	if enabled {
		ipsetMode, err := a.manager.IPSetMode()
		if err == nil {
			a.mu.Lock()
			a.cfg.GlobalIPSetMode = string(ipsetMode)
			a.mu.Unlock()
		}
	}
	return a.saveConfig()
}

func (a *App) SetWindowsAutostart(enabled bool) error {
	taskName := a.Config().ElevatedTaskName
	if taskName == "" {
		taskName = "ZapretTrayManager"
	}
	if enabled {
		if err := winexec.CreateAutostartTask(taskName); err != nil {
			return fmt.Errorf("failed to create autostart task: %w", err)
		}
		return nil
	}
	if err := winexec.DeleteAutostartTask(taskName); err != nil {
		return fmt.Errorf("failed to delete autostart task: %w", err)
	}
	return nil
}

func (a *App) WindowsAutostartEnabled() (bool, error) {
	taskName := a.Config().ElevatedTaskName
	if taskName == "" {
		taskName = "ZapretTrayManager"
	}
	return winexec.AutostartTaskExists(taskName)
}
