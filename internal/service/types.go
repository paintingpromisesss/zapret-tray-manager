package service

import "zapret-tray-manager/internal/winexec"

type State string

const (
	StateRunning      State = "running"
	StateStopped      State = "stopped"
	StateStartPending State = "start_pending"
	StateStopPending  State = "stop_pending"
	StateNotInstalled State = "not_installed"
	StateUnknown      State = "unknown"
)

type Status struct {
	Zapret            State
	WinDivert         State
	RunningStrategy   string
	InstalledStrategy string
	WinWSRunning      bool
}

type CommandRunner interface {
	Run(name string, args ...string) (winexec.Result, error)
}
