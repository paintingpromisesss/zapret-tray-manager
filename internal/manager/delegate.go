package manager

import (
	"zapret-tray-manager/internal/service"
	"zapret-tray-manager/internal/strategy"
)

func (m *Manager) ServiceStatus() (service.Status, error) {
	return m.svc.Status()
}

func (m *Manager) Start() error  { return m.svc.Start() }
func (m *Manager) Stop() error   { return m.svc.Stop() }
func (m *Manager) Remove() error { return m.svc.Remove() }

func (m *Manager) Install(s strategy.Strategy, autoRun bool) error {
	ports := m.GameFilterMode().Ports()
	return m.svc.Install(s, service.GameFilterPorts{All: ports.All, TCP: ports.TCP, UDP: ports.UDP}, autoRun)
}

func (m *Manager) ConfigureStrategy(s strategy.Strategy, autoRun bool, start bool) error {
	ports := m.GameFilterMode().Ports()
	return m.svc.Configure(s, service.GameFilterPorts{All: ports.All, TCP: ports.TCP, UDP: ports.UDP}, autoRun, start)
}

func (m *Manager) SetAutoRunService(autoRun bool) error {
	return m.svc.SetAutoRun(autoRun)
}

func (m *Manager) QueryAutoRunService() (bool, error) {
	return m.svc.QueryAutoRun()
}
