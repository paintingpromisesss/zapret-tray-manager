package service

import "errors"

var (
	ErrZapretServiceNotInstalled = errors.New("zapret service is not installed")
)

func (s *Service) Start() error {
	state, err := s.queryState("zapret")
	if err != nil {
		return err
	}
	if state == StateNotInstalled {
		return ErrZapretServiceNotInstalled
	}
	if state == StateRunning {
		return nil
	}
	return s.run("sc.exe", "start", "zapret")
}

func (s *Service) Stop() error {
	state, err := s.queryState("zapret")
	if err != nil {
		return err
	}
	if state == StateNotInstalled || state == StateStopped {
		return nil
	}

	var errs []error
	netStopErr := s.run("net.exe", "stop", "zapret")

	if running, err := s.isProcessRunning("winws.exe"); err != nil {
		errs = append(errs, err)
	} else if running {
		if err := s.run("taskkill.exe", "/IM", "winws.exe", "/F"); err != nil {
			errs = append(errs, err)
		}
	}

	if netStopErr != nil {
		if isBenignStopError(netStopErr) {
			s.logger.Info("ignoring benign service stop error", "service", "zapret", "error", netStopErr)
		} else if finalState, err := s.queryState("zapret"); err != nil || (finalState != StateStopped && finalState != StateNotInstalled) {
			errs = append(errs, netStopErr)
		} else {
			s.logger.Info("net stop reported an error but service is stopped", "service", "zapret", "error", netStopErr)
		}
	}

	return errors.Join(errs...)
}

func (s *Service) Remove() error {
	var errs []error

	if err := s.stopAndDelete("zapret"); err != nil {
		errs = append(errs, err)
	}

	if running, err := s.isProcessRunning("winws.exe"); err != nil {
		errs = append(errs, err)
	} else if running {
		if err := s.run("taskkill.exe", "/IM", "winws.exe", "/F"); err != nil {
			errs = append(errs, err)
		}
	}

	if err := s.stopAndDelete("WinDivert"); err != nil {
		errs = append(errs, err)
	}
	if err := s.stopAndDelete("WinDivert14"); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
