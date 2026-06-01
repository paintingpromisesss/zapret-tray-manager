//go:build windows

package winexec

import (
	"golang.org/x/sys/windows"
)

func IsAdmin() (bool, error) {
	return windows.GetCurrentProcessToken().IsElevated(), nil
}
