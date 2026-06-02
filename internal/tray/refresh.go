package tray

import (
	"time"

	"fyne.io/systray"

	"zapret-tray-manager/internal/i18n"
	"zapret-tray-manager/internal/service"
	"zapret-tray-manager/internal/zapretver"
)

func (t *Tray) refresh() {
	if t.app.Busy() {
		t.requestRefresh(2 * time.Second)
		return
	}

	in := t.collectState()

	t.statusMu.Lock()
	t.lastStatus = in.status
	t.statusMu.Unlock()

	fp := in.fingerprint()
	if t.hasFingerprint && fp == t.lastFingerprint {
		// Controlled state is identical to the last render; nothing in the menu
		// would change. Skip the (otherwise unconditional) item updates.
		t.scheduleAutoRefresh()
		return
	}
	t.lastFingerprint = fp
	t.hasFingerprint = true

	t.applyStatus(in)
	t.scheduleAutoRefresh()
}

// collectState gathers every input that the menu is rendered from, including
// the live queries (autostart, release download state, local roots) so they
// are read exactly once per refresh and can be both fingerprinted and applied.
func (t *Tray) collectState() stateInputs {
	status := t.app.Refresh()

	autostartEnabled, autostartErr := t.app.WindowsAutostartEnabled()
	if autostartErr != nil {
		t.logger.Warn("could not query Windows autostart state", "error", autostartErr)
	}

	releases := t.versionReleases
	if len(releases) > maxReleaseItems {
		releases = releases[:maxReleaseItems]
	}
	downloaded := make([]bool, len(releases))
	releaseRoots := make([]string, len(releases))
	localRoots := t.app.LocalZapretRoots()
	localVersions := make(map[string]struct{}, len(localRoots))
	for _, r := range localRoots {
		localVersions[zapretver.NormalizeVersion(r.Version)] = struct{}{}
	}
	for i, r := range releases {
		releaseRoots[i] = t.app.ReleaseRootPath(r)
		_, isLocal := localVersions[zapretver.NormalizeVersion(r.Version)]
		downloaded[i] = isLocal || t.app.IsReleaseDownloaded(r)
	}

	return stateInputs{
		status:           status,
		cfg:              t.app.Config(),
		autostartEnabled: autostartEnabled,
		autostartErr:     autostartErr != nil,
		releases:         releases,
		localRoots:       localRoots,
		userRoots:        t.app.UserLocalRoots(),
		downloaded:       downloaded,
		releaseRoots:     releaseRoots,
		busy:             t.app.Busy(),
		pendingUpdateVer: t.pendingUpdateVer,
	}
}

//nolint:gocyclo,cyclop,funlen // Maps many independent status fields onto menu items; flat branching is the clearest form.
func (t *Tray) applyStatus(in stateInputs) {
	status := in.status
	cfg := in.cfg

	if status.Valid {
		t.errorItem.SetTitle(t.s.ErrorNone)
		t.errorItem.Hide()
	} else {
		t.errorItem.SetTitle(t.s.ErrorPrefix + trimMenuText(t.errorText(status.ValidationError)))
		t.errorItem.Show()
		if t.lastStatusValid {
			root := t.app.Config().CurrentRoot
			detail := t.errorText(status.ValidationError)
			go errorDialog(i18n.AppTitle, t.s.ZapretFolderNotFound, root+"\n\n"+detail)
		}
	}
	t.lastStatusValid = status.Valid

	t.serviceItem.SetTitle(t.s.ServicePrefix + serviceTitle(status.ServiceStatus.Zapret))
	t.strategyItem.SetTitle(t.s.StrategyPrefix + fallback(status.ServiceStatus.InstalledStrategy, t.s.StrategyNone))
	t.gameItem.SetTitle("Game filter: " + gameFilterTitle(status.GameFilterMode))
	if status.IPSetError != nil {
		t.ipsetItem.SetTitle("IPSet: error")
	} else {
		t.ipsetItem.SetTitle("IPSet: " + ipsetTitle(status.IPSetMode))
	}
	systray.SetTooltip("Zapret Tray Manager\n" +
		serviceTitle(status.ServiceStatus.Zapret) + "\n" +
		fallback(status.ServiceStatus.InstalledStrategy, t.s.StrategyNone))

	if cfg.ZapretAutoRunEnabled {
		t.autoRunItem.Check()
	} else {
		t.autoRunItem.Uncheck()
	}

	if !in.autostartErr {
		if in.autostartEnabled {
			t.autostartItem.Check()
		} else {
			t.autostartItem.Uncheck()
		}
	}

	if cfg.GlobalSettingsEnabled {
		t.globalSettingsItem.Check()
	} else {
		t.globalSettingsItem.Uncheck()
	}

	if cfg.VPNManageEnabled {
		t.vpnManageItem.Check()
	} else {
		t.vpnManageItem.Uncheck()
	}

	for mode, item := range t.gameItems {
		if mode == status.GameFilterMode {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
	for mode, item := range t.ipsetItems {
		if mode == status.IPSetMode && status.IPSetError == nil {
			item.Check()
		} else {
			item.Uncheck()
		}
	}

	t.updateStrategies(status)
	t.updateZapretVersions(t.versionReleases, status.Valid)
	t.updateLocalZapretMenu(status.Valid)
	if status.ServiceStatus.Zapret == service.StateRunning || status.ServiceStatus.Zapret == service.StateStartPending {
		t.runStopItem.SetTitle(t.s.Stop)
		systray.SetIcon(iconRunning)
	} else {
		t.runStopItem.SetTitle(t.s.Run)
		systray.SetIcon(iconStopped)
	}

	enabled := status.Valid && !t.app.Busy()
	t.setActionItemsEnabled(enabled)
	if enabled && len(status.Strategies) > 0 {
		t.strategiesRoot.Enable()
	} else {
		t.strategiesRoot.Disable()
	}
	serviceInstalled := status.ServiceStatus.Zapret != service.StateNotInstalled
	if !serviceInstalled {
		t.runStopItem.Disable()
	}
	if serviceInstalled && !t.app.Busy() {
		t.removeItem.Enable()
	} else {
		t.removeItem.Disable()
	}
	t.setGlobalItemsEnabled(!t.app.Busy())
}

func (t *Tray) requestRefresh(delay time.Duration) {
	t.timerMu.Lock()
	if t.timer != nil {
		t.timer.Stop()
	}
	t.timer = time.AfterFunc(delay, t.refresh)
	t.timerMu.Unlock()
}

func (t *Tray) scheduleAutoRefresh() {
	cfg := t.app.Config()
	interval, err := cfg.StateRefreshIntervalDuration()
	if err != nil || interval <= 0 {
		return
	}
	t.requestRefresh(interval)
}
