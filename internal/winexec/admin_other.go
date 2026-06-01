//go:build !windows

package winexec

func IsAdmin() (bool, error) {
	return true, nil
}
