//go:build !windows

package winexec

func CreateAutostartTask(_ string) error         { return nil }
func DeleteAutostartTask(_ string) error         { return nil }
func AutostartTaskExists(_ string) (bool, error) { return false, nil }
