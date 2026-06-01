package tray

import (
	"log/slog"
	"sync"
	"time"

	"fyne.io/systray"

	"zapret-tray-manager/internal/app"
	"zapret-tray-manager/internal/i18n"
	"zapret-tray-manager/internal/manager"
	"zapret-tray-manager/internal/zapretver"
)

const (
	maxStrategyItems = 64
	maxReleaseItems  = 20
)

type Tray struct {
	app     *app.App
	logger  *slog.Logger
	logPath string
	s       *i18n.Strings

	serviceItem  *systray.MenuItem
	strategyItem *systray.MenuItem
	gameItem     *systray.MenuItem
	ipsetItem    *systray.MenuItem
	errorItem    *systray.MenuItem

	runStopItem     *systray.MenuItem
	removeItem      *systray.MenuItem
	updateIPSetItem *systray.MenuItem

	strategiesRoot        *systray.MenuItem
	strategyItems         []strategyItem
	customStrategyItems   []strategyItem
	addCustomStrategyItem *systray.MenuItem
	zapretVersionsRoot    *systray.MenuItem
	refreshReleasesItem   *systray.MenuItem
	githubItems           []releaseItem
	versionReleases       []zapretver.Release

	localZapretRoot  *systray.MenuItem
	userLocalItems   []userLocalItem
	addLocalRootItem *systray.MenuItem
	gameItems        map[manager.GameFilterMode]*systray.MenuItem
	ipsetItems       map[manager.IPSetMode]*systray.MenuItem

	settingsItem       *systray.MenuItem
	checkUpdatesItem   *systray.MenuItem
	pendingUpdateVer   string
	autoRunItem        *systray.MenuItem
	autostartItem      *systray.MenuItem
	globalSettingsItem *systray.MenuItem
	vpnStopItem        *systray.MenuItem
	vpnStartItem       *systray.MenuItem
	langEnItem         *systray.MenuItem
	langRuItem         *systray.MenuItem
	viewLogsItem       *systray.MenuItem
	exportLogsItem     *systray.MenuItem
	quitItem           *systray.MenuItem

	actionItems     []*systray.MenuItem
	statusMu        sync.Mutex
	lastStatus      manager.Status
	lastStatusValid bool

	timerMu sync.Mutex
	timer   *time.Timer
}

type strategyItem struct {
	item *systray.MenuItem
	name string
}

type releaseItem struct {
	item    *systray.MenuItem
	release zapretver.Release
	root    string
}

type userLocalItem struct {
	item *systray.MenuItem
	path string
}

func Run(application *app.App, logger *slog.Logger, logPath string) {
	if logger == nil {
		logger = slog.Default()
	}
	t := &Tray{
		app:        application,
		logger:     logger,
		logPath:    logPath,
		s:          i18n.Load(application.Config().Language),
		gameItems:  make(map[manager.GameFilterMode]*systray.MenuItem),
		ipsetItems: make(map[manager.IPSetMode]*systray.MenuItem),
	}
	systray.Run(t.onReady, t.onExit)
}

//nolint:funlen // One-time tray menu construction; linear item wiring reads better as a single block.
func (t *Tray) onReady() {
	systray.SetTitle("zapret")
	systray.SetTooltip("zapret-tray-manager")
	systray.SetIcon(iconStopped)

	t.serviceItem = disabledItem(t.s.ServiceUnknown)
	t.strategyItem = disabledItem(t.s.StrategyUnknown)
	t.gameItem = disabledItem("Game filter: unknown")
	t.ipsetItem = disabledItem("IPSet: unknown")
	t.errorItem = disabledItem(t.s.ErrorNone)
	t.errorItem.Hide()

	systray.AddSeparator()
	t.runStopItem = systray.AddMenuItem(t.s.Run, "")
	t.actionItems = append(t.actionItems, t.runStopItem)
	go t.listen(t.runStopItem, "Run/Stop", t.app.ToggleRunStop)

	systray.AddSeparator()
	t.strategiesRoot = systray.AddMenuItem(t.s.Strategies, "")
	t.strategiesRoot.Disable()

	t.initGameFilterMenu()
	t.initIPSetMenu()

	systray.AddSeparator()
	t.settingsItem = systray.AddMenuItem(t.s.Settings, "")

	t.zapretVersionsRoot = t.settingsItem.AddSubMenuItem(t.s.ZapretVersions, "")
	t.localZapretRoot = t.settingsItem.AddSubMenuItem(t.s.LocalZapret, "")

	t.settingsItem.AddSeparator()

	t.autostartItem = t.settingsItem.AddSubMenuItemCheckbox(t.s.AutostartWithWindows, "", false)
	t.autoRunItem = t.settingsItem.AddSubMenuItemCheckbox(t.s.AutoRunService, "", false)
	t.globalSettingsItem = t.settingsItem.AddSubMenuItemCheckbox(t.s.GlobalIPSetGameFilter, "", false)
	vpnInteractionItem := t.settingsItem.AddSubMenuItem(t.s.VPNInteraction, "")
	t.vpnStopItem = vpnInteractionItem.AddSubMenuItemCheckbox(t.s.VPNStopOnConnect, "", false)
	t.vpnStartItem = vpnInteractionItem.AddSubMenuItemCheckbox(t.s.VPNStartOnDisconnect, "", false)

	t.settingsItem.AddSeparator()

	logsItem := t.settingsItem.AddSubMenuItem(t.s.Logs, "")
	t.viewLogsItem = logsItem.AddSubMenuItem(t.s.ViewLogs, "")
	t.exportLogsItem = logsItem.AddSubMenuItem(t.s.ExportLogs, "")
	showServiceInfoItem := t.settingsItem.AddSubMenuItem(t.s.ShowServiceInfo, "")
	openFolderItem := t.settingsItem.AddSubMenuItem(t.s.OpenProgramFolder, "")
	langItem := t.settingsItem.AddSubMenuItem(t.s.LanguageMenu, "")
	cfg := t.app.Config()
	t.langEnItem = langItem.AddSubMenuItemCheckbox(t.s.LangEnglish, "", cfg.Language != "russian" && cfg.Language != "ru")
	t.langRuItem = langItem.AddSubMenuItemCheckbox(t.s.LangRussian, "", cfg.Language == "russian" || cfg.Language == "ru")
	go t.listenGlobal(t.langEnItem, "Set language English", func() error { return t.setLanguage("english") })
	go t.listenGlobal(t.langRuItem, "Set language Russian", func() error { return t.setLanguage("russian") })
	t.removeItem = t.settingsItem.AddSubMenuItem(t.s.RemoveServices, "")
	t.removeItem.Disable()

	t.actionItems = append(t.actionItems, t.autoRunItem, t.removeItem)
	go t.listenGlobal(t.autostartItem, "Autostart with Windows", t.toggleWindowsAutostart)
	go t.listen(t.autoRunItem, "Auto-run service", t.toggleAutoRun)
	go t.listenGlobal(t.globalSettingsItem, "Global IPSet/GameFilter", t.toggleGlobalSettings)
	go t.listenGlobal(t.vpnStopItem, "Stop when VPN connects", t.toggleVPNStop)
	go t.listenGlobal(t.vpnStartItem, "Start when VPN disconnects", t.toggleVPNStart)
	go t.listenGlobal(t.viewLogsItem, "View logs", t.viewLogs)
	go t.listenGlobal(t.exportLogsItem, "Export logs", t.exportLogs)
	go t.listenGlobal(showServiceInfoItem, "Show service info", t.showServiceInfo)
	go t.listenGlobal(openFolderItem, "Open program folder", t.openProgramFolder)
	go t.listen(t.removeItem, "Remove services", t.app.Remove)

	t.checkUpdatesItem = systray.AddMenuItem(t.updateCheckLabel(), "")
	go t.listenGlobal(t.checkUpdatesItem, "Check for updates", t.checkForUpdatesManual)

	systray.AddSeparator()
	t.quitItem = systray.AddMenuItem(t.s.Quit, "")
	go func() {
		<-t.quitItem.ClickedCh
		systray.Quit()
	}()

	t.app.SetVPNAfterAction(func() {
		t.requestRefresh(300 * time.Millisecond)
	})

	t.initVersionsMenu()
	t.refresh()
	go t.runGlobalAction("Refresh zapret versions", t.refreshZapretReleases)
	go t.checkForUpdatesSilent()
}

func (t *Tray) initGameFilterMenu() {
	gameRoot := systray.AddMenuItem("Game filter mode", "")
	for _, mode := range []manager.GameFilterMode{
		manager.GameFilterAll,
		manager.GameFilterTCP,
		manager.GameFilterUDP,
		manager.GameFilterDisabled,
	} {
		item := gameRoot.AddSubMenuItemCheckbox(gameFilterTitle(mode), "", false)
		t.gameItems[mode] = item
		t.actionItems = append(t.actionItems, item)
		go t.listen(item, "Set game filter", func() error {
			return t.app.SetGameFilter(mode)
		})
	}
}

func (t *Tray) initIPSetMenu() {
	ipsetRoot := systray.AddMenuItem("IPSet mode", "")
	for _, mode := range []manager.IPSetMode{manager.IPSetLoaded, manager.IPSetAny, manager.IPSetNone} {
		item := ipsetRoot.AddSubMenuItemCheckbox(ipsetTitle(mode), "", false)
		t.ipsetItems[mode] = item
		t.actionItems = append(t.actionItems, item)
		go t.listen(item, "Set IPSet", func() error {
			return t.app.SetIPSet(mode)
		})
	}
	t.updateIPSetItem = systray.AddMenuItem(t.s.UpdateIPSet, "")
	t.actionItems = append(t.actionItems, t.updateIPSetItem)
	go t.listen(t.updateIPSetItem, "Update IPSet", func() error {
		ctx, cancel := contextWithTimeout(45)
		defer cancel()
		return t.app.UpdateIPSet(ctx)
	})
}

func (t *Tray) onExit() {
	t.timerMu.Lock()
	if t.timer != nil {
		t.timer.Stop()
	}
	t.timerMu.Unlock()
}
