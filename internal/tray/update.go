package tray

import (
	"fmt"

	"zapret-tray-manager/internal/i18n"
	"zapret-tray-manager/internal/selfupdate"
)

// onCheckUpdatesClicked handles the menu item. When an update is already known
// (the item shows "Update available"), it downloads and launches the installer
// immediately. Otherwise it performs a check and reports the up-to-date result.
func (t *Tray) onCheckUpdatesClicked() error {
	if t.pendingUpdateVer != "" {
		t.installUpdate(t.pendingRelease)
		return nil
	}
	t.runUpdateCheck(true)
	return nil
}

// checkForUpdatesSilent is the startup background check. It only flags the menu
// item when a newer release is available; it never shows a dialog.
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

	if !check.Available {
		t.logger.Info("no update available", "current", check.Current)
		t.setUpdateAvailable("", selfupdate.Release{})
		if notifyWhenCurrent {
			current := fallback(check.Current, t.app.Version())
			infoDialog(i18n.AppTitle, fmt.Sprintf(t.s.UpdateUpToDateHeading, current), "")
		}
		return
	}

	t.logger.Info("update available", "current", check.Current, "latest", check.Release.Version)
	t.setUpdateAvailable(check.Release.TagName, check.Release)
}

func (t *Tray) installUpdate(release selfupdate.Release) {
	t.runGlobalAction("Download and install update", func() error {
		dlCtx, dlCancel := contextWithTimeout(180)
		defer dlCancel()
		return t.app.DownloadAndRunUpdate(dlCtx, release)
	})
}

func (t *Tray) setUpdateAvailable(version string, release selfupdate.Release) {
	t.pendingUpdateVer = version
	t.pendingRelease = release
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
