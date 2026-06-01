//go:build windows

package winexec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/text/encoding/charmap"
)

var (
	modkernel32         = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleOutCP = modkernel32.NewProc("GetConsoleOutputCP")
	procGetOEMCP        = modkernel32.NewProc("GetOEMCP")
)

func consoleOutputCP() uint32 {
	if r, _, _ := procGetConsoleOutCP.Call(); r != 0 { //nolint:errcheck // see comment
		return uint32(r) //nolint:gosec // codepage id fits in uint32
	}
	r, _, _ := procGetOEMCP.Call() //nolint:errcheck // see comment above
	return uint32(r)               //nolint:gosec // codepage id fits in uint32
}

func taskNotFoundError(raw []byte) bool {
	s := strings.ToLower(decodeOEM(raw))
	needles := []string{
		// English
		"cannot find",
		"does not exist",
		"the system cannot find",
		// Russian
		"не существует",
		"не найден",
		"не удается найти",
		"не удаётся найти",
		"указанный файл",
	}
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func decodeOEM(raw []byte) string {
	cm := charmapForCodePage(consoleOutputCP())
	if cm == nil {
		return string(raw)
	}
	decoded, err := cm.NewDecoder().Bytes(raw)
	if err != nil {
		return string(raw)
	}
	return string(decoded)
}

func charmapForCodePage(cp uint32) *charmap.Charmap {
	switch cp {
	case 866:
		return charmap.CodePage866
	case 437:
		return charmap.CodePage437
	case 850:
		return charmap.CodePage850
	case 1251:
		return charmap.Windows1251
	default:
		return nil
	}
}

func CreateAutostartTask(taskName string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	cmdLine := fmt.Sprintf(
		`schtasks.exe /Create /TN %s /TR "\"%s\"" /SC ONLOGON /RL HIGHEST /F`,
		quoteArg(taskName), exe,
	)
	cmd := &exec.Cmd{
		Path: `schtasks.exe`,
		SysProcAttr: &syscall.SysProcAttr{
			HideWindow: true,
			CmdLine:    cmdLine,
		},
	}
	if p, lookErr := exec.LookPath("schtasks.exe"); lookErr == nil {
		cmd.Path = p
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks create: %w: %s", err, strings.TrimSpace(decodeOEM(out)))
	}
	return nil
}

func quoteArg(s string) string {
	if strings.ContainsAny(s, " \t") {
		return `"` + s + `"`
	}
	return s
}

func DeleteAutostartTask(taskName string) error {
	//nolint:gosec,noctx // Fixed schtasks.exe, config-controlled task name; short-lived call.
	cmd := exec.Command("schtasks.exe", "/Delete", "/TN", taskName, "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if taskNotFoundError(out) {
			return nil
		}
		return fmt.Errorf("schtasks delete: %w: %s", err, strings.TrimSpace(decodeOEM(out)))
	}
	return nil
}

func AutostartTaskExists(taskName string) (bool, error) {
	//nolint:gosec,noctx // Fixed schtasks.exe, config-controlled task name; short-lived call.
	cmd := exec.Command("schtasks.exe", "/Query", "/TN", taskName)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if taskNotFoundError(out) {
			return false, nil
		}
		return false, fmt.Errorf("schtasks query: %w: %s", err, strings.TrimSpace(decodeOEM(out)))
	}
	return true, nil
}
