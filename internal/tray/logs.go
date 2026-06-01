//go:build windows

package tray

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"zapret-tray-manager/internal/winexec"
)

var (
	errSaveSelectionCanceled = errors.New("save selection canceled")
	ErrLogPathEmpty          = errors.New("log path is empty")
)

func (t *Tray) viewLogs() error {
	if t.logPath == "" {
		return ErrLogPathEmpty
	}
	if _, err := os.Stat(t.logPath); err != nil {
		return fmt.Errorf("log file does not exist: %w", err)
	}

	command := fmt.Sprintf("Get-Content -LiteralPath %s -Tail 200 -Wait", powerShellQuote(t.logPath))
	args := []string{
		"/c", "start", "", "powershell.exe",
		"-NoProfile", "-NoExit",
		"-Command", command,
	}
	t.logger.Info("view logs", "path", t.logPath)
	//nolint:gosec,noctx // Fixed cmd.exe, args from our quoted log path; detached fire-and-forget viewer.
	if err := exec.Command("cmd.exe", args...).Start(); err != nil {
		return fmt.Errorf("open log viewer: %w", err)
	}
	return nil
}

func (t *Tray) exportLogs() error {
	if t.logPath == "" {
		return ErrLogPathEmpty
	}

	target, err := pickSaveFile(filepath.Base(t.logPath))
	if errors.Is(err, errSaveSelectionCanceled) {
		t.logger.Info("export logs canceled")
		return nil
	}
	if err != nil {
		return err
	}

	if err := copyFile(t.logPath, target); err != nil {
		return err
	}
	t.logger.Info("logs exported", "source", t.logPath, "target", target)
	return nil
}

func pickSaveFile(defaultName string) (string, error) {
	const script = `
Add-Type -AssemblyName System.Windows.Forms
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$dialog = New-Object System.Windows.Forms.SaveFileDialog
$dialog.Title = 'Export zapret-tray-manager logs'
$dialog.Filter = 'Log files (*.log)|*.log|Text files (*.txt)|*.txt|All files (*.*)|*.*'
$dialog.FileName = $args[0]
if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
  Write-Output $dialog.FileName
  exit 0
}
exit 2
`
	output, err := winexec.Command("powershell.exe", "-NoProfile", "-STA", "-Command", script, defaultName).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 2 {
			return "", errSaveSelectionCanceled
		}
		return "", fmt.Errorf("save dialog: %w: %s", err, strings.TrimSpace(string(output)))
	}

	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return "", errSaveSelectionCanceled
	}
	return selected, nil
}

func copyFile(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath) //nolint:gosec // sourcePath is our own log file; targetPath is user-chosen save dialog.
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = source.Close() }()

	target, err := os.Create(targetPath) //nolint:gosec // see above
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}

	if _, err = io.Copy(target, source); err != nil {
		_ = target.Close()
		return fmt.Errorf("failed to copy file: %w", err)
	}
	if err := target.Close(); err != nil {
		return fmt.Errorf("failed to close target file: %w", err)
	}
	return nil
}

func pickBatFile() (string, error) {
	const script = `
Add-Type -AssemblyName System.Windows.Forms
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$dialog = New-Object System.Windows.Forms.OpenFileDialog
$dialog.Title = 'Select strategy file'
$dialog.Filter = 'Batch files (*.bat)|*.bat|All files (*.*)|*.*'
if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
  Write-Output $dialog.FileName
  exit 0
}
exit 2
`
	output, err := winexec.Command("powershell.exe", "-NoProfile", "-STA", "-Command", script).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 2 {
			return "", errSaveSelectionCanceled
		}
		return "", fmt.Errorf("open dialog: %w: %s", err, strings.TrimSpace(string(output)))
	}
	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return "", errSaveSelectionCanceled
	}
	return selected, nil
}

func powerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
