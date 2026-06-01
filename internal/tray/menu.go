package tray

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sqweek/dialog"

	"zapret-tray-manager/internal/i18n"
	"zapret-tray-manager/internal/manager"
	"zapret-tray-manager/internal/strategy"
	"zapret-tray-manager/internal/zapretver"
)

func (t *Tray) updateStrategies(status manager.Status) {
	var regular, custom []strategy.Strategy
	for _, s := range status.Strategies {
		if s.Custom {
			custom = append(custom, s)
		} else {
			regular = append(regular, s)
		}
	}

	for len(t.strategyItems) < len(regular) {
		t.addStrategyItem()
	}

	if t.addCustomStrategyItem == nil {
		t.strategiesRoot.AddSeparator()
		t.addCustomStrategyItem = t.strategiesRoot.AddSubMenuItem(t.s.AddCustomStrategy, "")
		go t.listenGlobal(t.addCustomStrategyItem, "Add custom strategy", t.pickAndAddCustomStrategy)
	}
	for len(t.customStrategyItems) < len(custom) {
		t.addCustomStrategyItem.Remove()
		t.addCustomStrategyItem = nil
		t.addCustomStrategySlot()
		t.addCustomStrategyItem = t.strategiesRoot.AddSubMenuItem(t.s.AddCustomStrategy, "")
		go t.listenGlobal(t.addCustomStrategyItem, "Add custom strategy", t.pickAndAddCustomStrategy)
	}

	t.renderRegular(regular, status.ServiceStatus.InstalledStrategy)
	t.renderCustom(custom, status.ServiceStatus.InstalledStrategy)
}

func (t *Tray) renderRegular(strategies []strategy.Strategy, installed string) {
	renderStrategySlots(t.strategyItems, strategies, installed)
}

func (t *Tray) renderCustom(strategies []strategy.Strategy, installed string) {
	renderStrategySlots(t.customStrategyItems, strategies, installed)
}

func renderStrategySlots(slots []strategyItem, strategies []strategy.Strategy, installed string) {
	for i := range slots {
		slots[i].name = ""
		slots[i].item.Hide()
	}
	for i, s := range strategies {
		if i >= len(slots) {
			break
		}
		base := strings.TrimSuffix(s.Name, filepath.Ext(s.Name))
		slots[i].name = s.Name
		checked := strings.EqualFold(base, installed) || strings.EqualFold(s.Name, installed)
		slots[i].item.SetTitle(s.Name)
		if checked {
			slots[i].item.Check()
		} else {
			slots[i].item.Uncheck()
		}
		slots[i].item.Show()
	}
}

func (t *Tray) addStrategyItem() {
	index := len(t.strategyItems)
	item := t.strategiesRoot.AddSubMenuItemCheckbox("", "", false)
	item.Hide()
	t.strategyItems = append(t.strategyItems, strategyItem{item: item})
	go t.listen(item, "Install strategy", func() error {
		name := t.strategyItems[index].name
		if name == "" {
			return nil
		}
		return t.app.InstallStrategy(name)
	})
}

func (t *Tray) addCustomStrategySlot() {
	index := len(t.customStrategyItems)
	item := t.strategiesRoot.AddSubMenuItemCheckbox("", "", false)
	item.Hide()
	t.customStrategyItems = append(t.customStrategyItems, strategyItem{item: item})
	go t.listen(item, "Install strategy", func() error {
		name := t.customStrategyItems[index].name
		if name == "" {
			return nil
		}
		return t.app.InstallStrategy(name)
	})
}

func (t *Tray) pickAndAddCustomStrategy() error {
	path, err := pickBatFile()
	if err != nil {
		if errors.Is(err, errSaveSelectionCanceled) {
			return nil
		}
		return err
	}
	if err := t.app.AddCustomStrategy(path); err != nil {
		return err
	}
	t.requestRefresh(300 * time.Millisecond)
	return nil
}

func (t *Tray) refreshZapretReleases() error {
	ctx, cancel := contextWithTimeout(35)
	defer cancel()

	releases, err := t.app.FetchZapretReleases(ctx, maxReleaseItems)
	if err != nil {
		return err
	}
	t.versionReleases = append([]zapretver.Release(nil), releases...)
	t.updateZapretVersions(releases, true)
	t.errorItem.SetTitle("Zapret versions: loaded")
	return nil
}

func (t *Tray) initVersionsMenu() {
	t.refreshReleasesItem = t.zapretVersionsRoot.AddSubMenuItem(t.s.RefreshVersions, "")
	go t.listenGlobal(t.refreshReleasesItem, "Refresh zapret versions", t.refreshZapretReleases)

	t.addLocalRootItem = t.localZapretRoot.AddSubMenuItem(t.s.AddLocalZapret, "")
	go t.listenGlobal(t.addLocalRootItem, "Add local zapret", t.pickAndAddLocalRoot)

	t.updateZapretVersions(nil, true)
	t.updateLocalZapretMenu(true)
}

func (t *Tray) updateZapretVersions(releases []zapretver.Release, rootValid bool) {
	cfg := t.app.Config()

	localVersions := make(map[string]struct{})
	for _, root := range t.app.LocalZapretRoots() {
		localVersions[zapretver.NormalizeVersion(root.Version)] = struct{}{}
	}

	filtered := releases
	if len(filtered) > maxReleaseItems {
		filtered = filtered[:maxReleaseItems]
	}

	for len(t.githubItems) < len(filtered) {
		t.allocGithubItem()
	}

	for i := range t.githubItems {
		t.githubItems[i].release = zapretver.Release{}
		t.githubItems[i].root = ""
		t.githubItems[i].item.Hide()
	}
	for i, release := range filtered {
		rootPath := t.app.ReleaseRootPath(release)
		_, isLocal := localVersions[zapretver.NormalizeVersion(release.Version)]
		downloaded := isLocal || t.app.IsReleaseDownloaded(release)
		t.githubItems[i].release = release
		t.githubItems[i].root = rootPath
		item := t.githubItems[i].item
		item.SetTitle(t.releaseTitle(release, downloaded))
		item.SetTooltip(fallback(release.ReleaseURL, release.AssetURL))
		setChecked(item, rootValid && samePath(rootPath, cfg.CurrentRoot))
		if release.AssetURL == "" {
			item.Disable()
		} else {
			item.Enable()
		}
		item.Show()
	}
}

func (t *Tray) allocGithubItem() {
	index := len(t.githubItems)
	item := t.zapretVersionsRoot.AddSubMenuItem("", "")
	item.Hide()
	t.githubItems = append(t.githubItems, releaseItem{item: item})
	go t.listenGlobal(item, "Download zapret version", func() error {
		release := t.githubItems[index].release
		if release.Version == "" {
			return nil
		}
		ctx, cancel := contextWithTimeout(75)
		defer cancel()
		if err := t.app.SwitchToZapretRelease(ctx, release); err != nil {
			return err
		}
		t.errorItem.SetTitle("Zapret root: " + trimMenuText(t.app.Config().CurrentRoot))
		t.requestRefresh(300 * time.Millisecond)
		return nil
	})
}

func (t *Tray) updateLocalZapretMenu(rootValid bool) {
	roots := t.app.UserLocalRoots()

	for len(t.userLocalItems) < len(roots) {
		t.addLocalRootItem.Remove()
		t.addLocalRootItem = nil
		t.allocUserLocalSlot()
		t.addLocalRootItem = t.localZapretRoot.AddSubMenuItem(t.s.AddLocalZapret, "")
		go t.listenGlobal(t.addLocalRootItem, "Add local zapret", t.pickAndAddLocalRoot)
	}

	for i := range t.userLocalItems {
		t.userLocalItems[i].path = ""
		t.userLocalItems[i].item.Hide()
	}

	exeLocalPaths := make(map[string]struct{})
	for _, r := range t.app.LocalZapretRoots() {
		exeLocalPaths[filepath.Clean(r.Path)] = struct{}{}
	}

	cfg := t.app.Config()
	for i, root := range roots {
		if i >= len(t.userLocalItems) {
			break
		}
		t.userLocalItems[i].path = root.Path
		item := t.userLocalItems[i].item
		item.SetTitle(root.Path)
		item.SetTooltip(root.Path)
		if root.Valid {
			item.Enable()
			_, isExeLocal := exeLocalPaths[filepath.Clean(root.Path)]
			setChecked(item, rootValid && !isExeLocal && samePath(root.Path, cfg.CurrentRoot))
		} else {
			item.Disable()
			item.Uncheck()
		}
		item.Show()
	}
}

func (t *Tray) allocUserLocalSlot() {
	index := len(t.userLocalItems)
	item := t.localZapretRoot.AddSubMenuItemCheckbox("", "", false)
	item.Hide()
	t.userLocalItems = append(t.userLocalItems, userLocalItem{item: item})
	go t.listenGlobal(item, "Switch local zapret", func() error {
		path := t.userLocalItems[index].path
		if path == "" {
			return nil
		}
		if _, err := t.app.FindZapretRoot(path); err != nil {
			//nolint:nilerr // Invalid local root is handled here (notify + remove); not an action failure.
			infoDialog(i18n.AppTitle, t.s.ZapretFolderNotFound, path)
			if err := t.app.RemoveUserLocalRoot(path); err != nil {
				t.logger.Warn("failed to remove invalid local root", "path", path, "error", err)
			}
			return nil
		}
		if err := t.app.SwitchToLocalRoot(path); err != nil {
			return err
		}
		t.requestRefresh(300 * time.Millisecond)
		return nil
	})
}

func (t *Tray) pickAndAddLocalRoot() error {
	cfg := t.app.Config()
	builder := dialog.Directory().Title("Choose zapret folder")
	if cfg.CurrentRoot != "" {
		builder.SetStartDir(cfg.CurrentRoot)
	}
	root, err := builder.Browse()
	if err != nil {
		if errors.Is(err, dialog.ErrCancelled) {
			return nil
		}
		return fmt.Errorf("choose zapret folder: %w", err)
	}
	if err := t.app.SwitchToLocalRoot(root); err != nil {
		return err
	}
	t.requestRefresh(300 * time.Millisecond)
	return nil
}
