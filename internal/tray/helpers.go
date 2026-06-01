package tray

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/systray"

	"zapret-tray-manager/internal/manager"
	"zapret-tray-manager/internal/service"
	"zapret-tray-manager/internal/zapretver"
)

func disabledItem(title string) *systray.MenuItem {
	item := systray.AddMenuItem(title, "")
	item.Disable()
	return item
}

func setChecked(item *systray.MenuItem, checked bool) {
	if checked {
		item.Check()
		return
	}
	item.Uncheck()
}

func samePath(left string, right string) bool {
	left = filepath.Clean(strings.TrimSpace(left))
	right = filepath.Clean(strings.TrimSpace(right))
	return strings.EqualFold(left, right)
}

func contextWithTimeout(seconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
}

func gameFilterTitle(mode manager.GameFilterMode) string {
	switch mode {
	case manager.GameFilterAll:
		return "All"
	case manager.GameFilterTCP:
		return "TCP"
	case manager.GameFilterUDP:
		return "UDP"
	default:
		return "Disabled"
	}
}

func ipsetTitle(mode manager.IPSetMode) string {
	switch mode {
	case manager.IPSetLoaded:
		return "Loaded"
	case manager.IPSetNone:
		return "None"
	case manager.IPSetAny:
		return "Any"
	default:
		return "Unknown"
	}
}

func serviceTitle(state service.State) string {
	if state == "" {
		return string(service.StateUnknown)
	}
	return string(state)
}

func (t *Tray) releaseTitle(release zapretver.Release, downloaded bool) string {
	version := fallback(release.Version, release.TagName)
	if downloaded {
		return version
	}
	if release.AssetURL == "" {
		return version + " " + t.s.OpenPage
	}
	return "⬇ " + version
}

func fallback(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

func (t *Tray) errorText(err error) string {
	if err == nil {
		return t.s.ErrorNone
	}
	if errors.Is(err, manager.ErrPathNotFound) || errors.Is(err, manager.ErrRootEmpty) {
		return t.s.ZapretFolderNotFound
	}
	return err.Error()
}

func trimMenuText(text string) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", "; "))
	if len(text) <= 120 {
		return text
	}
	return text[:117] + "..."
}
