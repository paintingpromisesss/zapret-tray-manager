//go:build !windows

package tray

import "errors"

var (
	errSaveSelectionCanceled = errors.New("save selection canceled")
	ErrLogPathEmpty          = errors.New("log path is empty")
)

func (t *Tray) viewLogs() error {
	return errors.New("not supported on this platform")
}

func (t *Tray) exportLogs() error {
	return errors.New("not supported on this platform")
}

func pickBatFile() (string, error) {
	return "", errors.New("not supported on this platform")
}
