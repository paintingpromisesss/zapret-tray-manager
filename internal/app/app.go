package app

import (
	"errors"
	"log/slog"
	"sync"

	"zapret-tray-manager/internal/config"
	"zapret-tray-manager/internal/manager"
	"zapret-tray-manager/internal/selfupdate"
	"zapret-tray-manager/internal/strategy"
	"zapret-tray-manager/internal/winexec"
	"zapret-tray-manager/internal/zapretver"
)

var (
	ErrBusy               = errors.New("action already running")
	ErrStrategyNotFound   = errors.New("strategy not found")
	ErrZapretRootNotFound = errors.New("zapret root not found")
)

type LocalZapretRoot struct {
	Version string
	Path    string
}

type App struct {
	store          *config.Store
	cfg            *config.Config
	manager        *manager.Manager
	verClient      *zapretver.Client
	updateClient   *selfupdate.Client
	version        string
	logger         *slog.Logger
	mu             sync.Mutex
	busy           bool
	tunWatcher     *winexec.TunWatcher
	vpnAfterAction func()
	// stoppedByVPN records that zapret was running and we stopped it on a VPN
	// connect, so that on VPN disconnect we know to restart it. If zapret was
	// already stopped when the VPN connected, this stays false and disconnect
	// does nothing. Guarded by mu.
	stoppedByVPN bool
}

func New(
	store *config.Store,
	cfg *config.Config,
	mgr *manager.Manager,
	vc *zapretver.Client,
	uc *selfupdate.Client,
	version string,
	logger *slog.Logger,
) *App {
	if cfg == nil {
		cfg = config.Default()
	}
	if logger == nil {
		logger = slog.Default()
	}
	a := &App{
		store:        store,
		cfg:          cfg,
		manager:      mgr,
		verClient:    vc,
		updateClient: uc,
		version:      version,
		logger:       logger,
	}
	a.syncTunWatcher()
	return a
}

func (a *App) Version() string {
	return a.version
}

func (a *App) Config() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return *a.cfg
}

func (a *App) Busy() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.busy
}

func (a *App) RunExclusive(label string, action func() error) error {
	a.mu.Lock()
	if a.busy {
		a.mu.Unlock()
		return ErrBusy
	}
	a.busy = true
	a.mu.Unlock()

	a.logger.Info("action start", "label", label)
	defer func() {
		a.mu.Lock()
		a.busy = false
		a.mu.Unlock()
		a.logger.Info("action done", "label", label)
	}()

	if err := action(); err != nil {
		a.logger.Error("action failed", "label", label, "error", err)
		return err
	}
	return nil
}

func (a *App) Refresh() manager.Status {
	customs := a.CustomStrategies()
	status := a.manager.Status()
	strategies, err := strategy.List(a.manager.Root, customs, CustomStrategiesDir())
	if err != nil {
		status.StrategiesError = err
	} else {
		status.Strategies = strategies
	}
	return status
}

func (a *App) saveConfig() error {
	a.mu.Lock()
	cfg := *a.cfg
	a.mu.Unlock()
	if a.store == nil {
		return nil
	}
	return a.store.Write(&cfg)
}
