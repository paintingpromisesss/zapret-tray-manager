package tray

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"fyne.io/systray"

	"zapret-tray-manager/internal/app"
)

func (t *Tray) listen(item *systray.MenuItem, label string, action func() error) {
	for range item.ClickedCh {
		t.runAction(label, action)
	}
}

func (t *Tray) listenGlobal(item *systray.MenuItem, label string, action func() error) {
	for range item.ClickedCh {
		t.runGlobalAction(label, action)
	}
}

func (t *Tray) runAction(label string, action func() error) {
	go func() {
		t.setActionItemsEnabled(false)
		t.errorItem.SetTitle("Working: " + trimMenuText(label))
		t.errorItem.Show()
		err := t.app.RunExclusive(label, action)
		if err != nil {
			if errors.Is(err, app.ErrBusy) {
				t.errorItem.SetTitle("Working: busy")
				t.requestRefresh(time.Second)
				return
			}
			t.logger.Error("tray action failed", "label", label, "error", err)
			t.errorItem.SetTitle("Error: " + trimMenuText(err.Error()))
			t.setActionItemsEnabled(true)
			t.requestRefresh(500 * time.Millisecond)
			return
		}
		t.errorItem.SetTitle("Error: none")
		t.errorItem.Hide()
		t.requestRefresh(300 * time.Millisecond)
	}()
}

func (t *Tray) runGlobalAction(label string, action func() error) {
	go func() {
		t.setGlobalItemsEnabled(false)
		t.errorItem.SetTitle("Working: " + trimMenuText(label))
		t.errorItem.Show()
		err := t.app.RunExclusive(label, action)
		if err != nil {
			if errors.Is(err, app.ErrBusy) {
				t.errorItem.SetTitle("Working: busy")
				t.setGlobalItemsEnabled(true)
				t.requestRefresh(time.Second)
				return
			}
			t.logger.Error("tray action failed", "label", label, "error", err)
			t.errorItem.SetTitle("Error: " + trimMenuText(err.Error()))
			t.setGlobalItemsEnabled(true)
			t.requestRefresh(500 * time.Millisecond)
			return
		}
		t.errorItem.SetTitle("Error: none")
		t.errorItem.Hide()
		t.setGlobalItemsEnabled(true)
		t.requestRefresh(300 * time.Millisecond)
	}()
}

func (t *Tray) setActionItemsEnabled(enabled bool) {
	for _, item := range t.actionItems {
		if enabled {
			item.Enable()
		} else {
			item.Disable()
		}
	}
}

func (t *Tray) setGlobalItemsEnabled(enabled bool) {
	if enabled {
		if t.refreshReleasesItem != nil {
			t.refreshReleasesItem.Enable()
		}
		for _, item := range t.githubItems {
			if item.release.Version != "" && item.release.AssetURL != "" {
				item.item.Enable()
			}
		}
		return
	}
	if t.refreshReleasesItem != nil {
		t.refreshReleasesItem.Disable()
	}
	for _, item := range t.githubItems {
		item.item.Disable()
	}
}

func (t *Tray) setLanguage(lang string) error {
	if err := t.app.SetLanguage(lang); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	//nolint:gosec,noctx // exe is our own resolved path; fire-and-forget re-launch, process exits right after.
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart for language change: %w", err)
	}
	systray.Quit()
	os.Exit(0)
	return nil
}

func (t *Tray) toggleAutoRun() error {
	cfg := t.app.Config()
	return t.app.SetAutoRunService(!cfg.ZapretAutoRunEnabled)
}

func (t *Tray) toggleGlobalSettings() error {
	cfg := t.app.Config()
	enabled := !cfg.GlobalSettingsEnabled
	if err := t.app.SetGlobalSettingsEnabled(enabled); err != nil {
		return err
	}
	if enabled {
		t.globalSettingsItem.Check()
	} else {
		t.globalSettingsItem.Uncheck()
	}
	return nil
}

func (t *Tray) toggleVPNStop() error {
	cfg := t.app.Config()
	enabled := !cfg.VPNStopOnConnect
	if err := t.app.SetVPNStopOnConnect(enabled); err != nil {
		return err
	}
	if enabled {
		t.vpnStopItem.Check()
	} else {
		t.vpnStopItem.Uncheck()
	}
	return nil
}

func (t *Tray) toggleVPNStart() error {
	cfg := t.app.Config()
	enabled := !cfg.VPNStartOnDisconnect
	if err := t.app.SetVPNStartOnDisconnect(enabled); err != nil {
		return err
	}
	if enabled {
		t.vpnStartItem.Check()
	} else {
		t.vpnStartItem.Uncheck()
	}
	return nil
}

func (t *Tray) toggleWindowsAutostart() error {
	enabled, err := t.app.WindowsAutostartEnabled()
	if err != nil {
		return err
	}
	if err := t.app.SetWindowsAutostart(!enabled); err != nil {
		return err
	}
	if !enabled {
		t.autostartItem.Check()
	} else {
		t.autostartItem.Uncheck()
	}
	return nil
}
