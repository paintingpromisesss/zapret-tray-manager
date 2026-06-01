package manager

import (
	"errors"
	"log/slog"
	"path/filepath"

	"zapret-tray-manager/internal/client"
	"zapret-tray-manager/internal/service"
	"zapret-tray-manager/internal/strategy"
	"zapret-tray-manager/internal/winexec"
)

var ErrRootEmpty = errors.New("root path is empty")

type Status struct {
	Strategies []strategy.Strategy

	GameFilterMode     GameFilterMode
	IPSetMode          IPSetMode
	ValidationError    error
	StrategiesError    error
	ServiceStatusError error
	IPSetError         error
	AdminCheckError    error
	ServiceStatus      service.Status
	Valid              bool
	Admin              bool
}

type Manager struct {
	Logger     *slog.Logger
	httpClient *client.Client
	svc        *service.Service
	Root       string
}

func New(root string, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	root = normalizeRoot(root)
	return &Manager{
		Root:       root,
		Logger:     logger,
		httpClient: client.NewClient(),
		svc:        service.New(root, logger),
	}
}

func (m *Manager) SetRoot(root string) {
	m.Root = normalizeRoot(root)
	m.svc.SetRoot(m.Root)
}

func (m *Manager) Status() Status {
	status := Status{
		GameFilterMode: m.GameFilterMode(),
	}

	if err := m.Validate(); err != nil {
		status.ValidationError = err
	} else {
		status.Valid = true
	}

	strategies, err := strategy.List(m.Root, nil, "")
	if err != nil {
		status.StrategiesError = err
	} else {
		status.Strategies = strategies
	}

	ipsetMode, err := m.IPSetMode()
	if err != nil {
		status.IPSetError = err
	} else {
		status.IPSetMode = ipsetMode
	}

	admin, err := winexec.IsAdmin()
	if err != nil {
		status.AdminCheckError = err
	}
	status.Admin = admin

	svcStatus, err := m.svc.Status()
	if err != nil {
		status.ServiceStatusError = err
	}
	status.ServiceStatus = svcStatus

	return status
}

func normalizeRoot(root string) string {
	if root == "" {
		return ""
	}
	return filepath.Clean(root)
}
