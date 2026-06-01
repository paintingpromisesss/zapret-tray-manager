//go:build !windows

package tray

import "errors"

func (t *Tray) openProgramFolder() error {
	return errors.New("not supported on this platform")
}

func (t *Tray) showServiceInfo() error {
	return errors.New("not supported on this platform")
}
