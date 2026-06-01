//go:build windows

package tray

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

func (t *Tray) openProgramFolder() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	//nolint:gosec,noctx // Fixed explorer.exe, our own dir; detached fire-and-forget launch.
	if err := exec.Command("explorer.exe", filepath.Dir(exe)).Start(); err != nil {
		return fmt.Errorf("open program folder: %w", err)
	}
	return nil
}

func psEncoded(script string) string {
	runes := utf16.Encode([]rune(script))
	buf := make([]byte, len(runes)*2)
	for i, r := range runes {
		//nolint:gosec // Deliberate little-endian split of a uint16 into two bytes.
		buf[i*2] = byte(r)
		buf[i*2+1] = byte(r >> 8)
	}
	return base64.StdEncoding.EncodeToString(buf)
}

func (t *Tray) showServiceInfo() error {
	ps := `
$sep = '-' * 52

function Get-SvcState($name) {
    $s = Get-Service -Name $name -ErrorAction SilentlyContinue
    if (-not $s) { return 'не установлена' }
    switch ($s.Status) {
        'Running' { 'запущена' }
        'Stopped' { 'остановлена' }
        default   { $s.Status }
    }
}

function Get-StartType($name) {
    $v = (Get-ItemProperty "HKLM:\System\CurrentControlSet\Services\$name" -ErrorAction SilentlyContinue).Start
    switch ($v) {
        2 { 'автозапуск' }
        3 { 'вручную' }
        4 { 'отключена' }
        default { "неизвестно ($v)" }
    }
}

$zapState  = Get-SvcState  'zapret'
$zapStart  = Get-StartType 'zapret'
$wdState   = Get-SvcState  'WinDivert'

$svcReg = Get-ItemProperty "HKLM:\System\CurrentControlSet\Services\zapret" -ErrorAction SilentlyContinue

$strategy = $svcReg.'zapret-discord-youtube'
if (-not $strategy) { $strategy = 'неизвестна' }

$imagePath = $svcReg.ImagePath
$workDir = '—'
if ($imagePath) {
    if ($imagePath -match '^"([^"]+)"') {
        $workDir = Split-Path $Matches[1] -Parent
    } elseif ($imagePath -match '^(\S+)') {
        $workDir = Split-Path $Matches[1] -Parent
    }
}

Write-Host $sep
Write-Host '  Служба zapret'
Write-Host $sep
Write-Host "  Состояние   : $zapState"
Write-Host "  Тип запуска : $zapStart"
Write-Host "  Стратегия   : $strategy"
Write-Host "  Рабочая папка: $workDir"
Write-Host ''
Write-Host $sep
Write-Host '  Служба WinDivert'
Write-Host $sep
Write-Host "  Состояние   : $wdState"
Write-Host $sep
Write-Host ''
Write-Host 'Нажмите любую клавишу для закрытия...'
$null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')
`
	encoded := psEncoded(ps)
	params, err := windows.UTF16PtrFromString(`/c powershell -NoProfile -ExecutionPolicy Bypass -EncodedCommand ` + encoded)
	if err != nil {
		return fmt.Errorf("encode command params: %w", err)
	}
	exe, err := windows.UTF16PtrFromString("cmd.exe")
	if err != nil {
		return fmt.Errorf("encode exe path: %w", err)
	}
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return fmt.Errorf("encode shell verb: %w", err)
	}
	if err := windows.ShellExecute(0, verb, exe, params, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("show service info window: %w", err)
	}
	return nil
}
