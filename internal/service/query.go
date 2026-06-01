package service

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInstalledStrategyRegistryValueNotFound = errors.New("installed strategy registry value not found")
)

func (s *Service) Status() (Status, error) {
	var status Status
	var errs []error

	zapretState, err := s.queryState("zapret")
	if err != nil {
		errs = append(errs, err)
	}
	status.Zapret = zapretState

	winDivertState, err := s.queryState("WinDivert")
	if err != nil {
		errs = append(errs, err)
	}
	status.WinDivert = winDivertState

	winWSRunning, err := s.isProcessRunning("winws.exe")
	if err != nil {
		errs = append(errs, err)
	}
	status.WinWSRunning = winWSRunning

	strategy, err := s.queryInstalledStrategy()
	if err == nil {
		status.InstalledStrategy = strategy
		status.RunningStrategy = strategy
	} else if status.Zapret != StateNotInstalled {
		errs = append(errs, err)
	}

	return status, errors.Join(errs...)
}

func (s *Service) queryState(name string) (State, error) {
	result, err := s.runner.Run("sc.exe", "query", name)
	if err != nil {
		if strings.Contains(err.Error(), "1060") {
			return StateNotInstalled, nil
		}
		return StateUnknown, fmt.Errorf("failed to query service state: %w", err)
	}
	state, ok := parseState(result.Output)
	if !ok {
		return StateUnknown, nil
	}
	return state, nil
}

func parseState(output string) (State, bool) {
	upper := strings.ToUpper(output)
	switch {
	case strings.Contains(upper, "RUNNING"):
		return StateRunning, true
	case strings.Contains(upper, "START_PENDING"):
		return StateStartPending, true
	case strings.Contains(upper, "STOP_PENDING"):
		return StateStopPending, true
	case strings.Contains(upper, "STOPPED"):
		return StateStopped, true
	}
	// STATE line found but status unrecognized (e.g. PAUSED)
	for _, line := range strings.Split(output, "\n") {
		upperLine := strings.ToUpper(strings.TrimSpace(line))
		if strings.Contains(upperLine, "STATE") && strings.Contains(upperLine, ":") {
			return StateUnknown, true
		}
	}
	return StateUnknown, false
}

func (s *Service) isProcessRunning(imageName string) (bool, error) {
	result, err := s.runner.Run("tasklist.exe", "/FI", "IMAGENAME eq "+imageName)
	if err != nil {
		return false, fmt.Errorf("failed to query running processes: %w", err)
	}
	return strings.Contains(strings.ToLower(result.Output), strings.ToLower(imageName)), nil
}

func (s *Service) queryInstalledStrategy() (string, error) {
	result, err := s.runner.Run(
		"reg.exe",
		"query",
		`HKLM\System\CurrentControlSet\Services\zapret`,
		"/v",
		"zapret-discord-youtube",
	)
	if err != nil {
		return "", fmt.Errorf("failed to query installed strategy: %w", err)
	}

	for _, line := range strings.Split(result.Output, "\n") {
		if value, ok := parseRegValue(line, "zapret-discord-youtube"); ok {
			return value, nil
		}
	}
	return "", ErrInstalledStrategyRegistryValueNotFound
}

func parseRegValue(line, name string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(name)) {
		return "", false
	}
	rest := strings.TrimSpace(trimmed[len(name):])
	for _, regType := range []string{"REG_SZ", "REG_EXPAND_SZ", "REG_MULTI_SZ", "REG_DWORD", "REG_QWORD", "REG_BINARY", "REG_NONE"} {
		if strings.HasPrefix(strings.ToUpper(rest), regType) {
			return strings.TrimSpace(rest[len(regType):]), true
		}
	}
	return "", false
}

func (s *Service) QueryAutoRun() (bool, error) {
	result, err := s.runner.Run(
		"reg.exe", "query",
		`HKLM\System\CurrentControlSet\Services\zapret`,
		"/v", "Start",
	)
	if err != nil {
		//nolint:nilerr // reg query fails when the service is absent, which means auto-run is off.
		return false, nil
	}
	for _, line := range strings.Split(result.Output, "\n") {
		if value, ok := parseRegValue(line, "Start"); ok {
			return value == "0x2", nil
		}
	}
	return false, nil
}

func (s *Service) stopAndDelete(name string) error {
	state, err := s.queryState(name)
	if err != nil {
		return err
	}
	if state == StateNotInstalled {
		return nil
	}

	if state == StateRunning || state == StateStartPending || state == StateStopPending {
		if err := s.run("net.exe", "stop", name); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
	}

	if err := s.run("sc.exe", "delete", name); err != nil {
		if isBenignDeleteError(err) {
			s.logger.Info("ignoring benign service delete error", "service", name, "error", err)
			return nil
		}
		return err
	}
	return nil
}
