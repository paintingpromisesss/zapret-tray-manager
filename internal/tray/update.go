package tray

import (
	"fmt"

	"zapret-tray-manager/internal/i18n"
)

func (t *Tray) checkForUpdatesManual() error {
	t.runUpdateCheck(true)
	return nil
}

func (t *Tray) checkForUpdatesSilent() {
	t.runUpdateCheck(false)
}

func (t *Tray) runUpdateCheck(notifyWhenCurrent bool) {
	ctx, cancel := contextWithTimeout(35)
	defer cancel()

	check, err := t.app.CheckForUpdate(ctx)
	if err != nil {
		t.logger.Warn("update check failed", "error", err)
		if notifyWhenCurrent {
			errorDialog(i18n.AppTitle, "", t.errorText(err))
		}
		return
	}

	current := fallback(check.Current, t.app.Version())

	if !check.Available {
		t.logger.Info("no update available", "current", check.Current)
		t.setUpdateAvailable("")
		if notifyWhenCurrent {
			infoDialog(
				i18n.AppTitle,
				t.s.UpdateUpToDateHeading,
				fmt.Sprintf(t.s.UpdateUpToDateBody, current),
			)
		}
		return
	}

	t.logger.Info("update available", "current", check.Current, "latest", check.Release.Version)
	t.setUpdateAvailable(check.Release.TagName)
	confirmed := confirmDialog(
		i18n.AppTitle,
		fmt.Sprintf(t.s.UpdateAvailableHeading, check.Release.Version),
		fmt.Sprintf(t.s.UpdateAvailableBody, current),
	)
	if !confirmed {
		return
	}

	t.runGlobalAction("Download and install update", func() error {
		dlCtx, dlCancel := contextWithTimeout(180)
		defer dlCancel()
		return t.app.DownloadAndRunUpdate(dlCtx, check.Release)
	})
}

func (t *Tray) setUpdateAvailable(version string) {
	t.pendingUpdateVer = version
	if t.checkUpdatesItem != nil {
		t.checkUpdatesItem.SetTitle(t.updateCheckLabel())
	}
}
func (t *Tray) updateCheckLabel() string {
	if t.pendingUpdateVer != "" {
		return fmt.Sprintf(t.s.UpdateAvailableMenu, t.pendingUpdateVer)
	}
	return t.s.CheckForUpdates
}
