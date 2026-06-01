package strategy_test

import (
	"os"
	"path/filepath"
	"testing"

	"zapret-tray-manager/internal/strategy"
)

func TestParseStrategyResolvesVariablesAndContinuation(t *testing.T) {
	t.Parallel()
	root := newTestRoot(t)
	content := `@echo off
set "BIN=%~dp0bin\"
start "zapret: general" /min "%BIN%winws.exe" --wf-tcp=80,%GameFilterTCP% ^
--hostlist="%LISTS%list-general.txt" --dpi-desync-fake-tls="%BIN%tls.bin"
`
	strategyPath := filepath.Join(root, "general.bat")
	if err := os.WriteFile(strategyPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	ports := strategy.GameFilterPorts{All: "0-65535", TCP: "1024-65535", UDP: "1024-65535"}
	parsed, err := strategy.Parse(strategy.Strategy{Name: "general.bat", Path: strategyPath}, root, ports)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	want := []string{
		"--wf-tcp=80,1024-65535",
		"--hostlist=" + filepath.Join(root, "lists") + string(os.PathSeparator) + "list-general.txt",
		"--dpi-desync-fake-tls=" + filepath.Join(root, "bin") + string(os.PathSeparator) + "tls.bin",
	}
	if len(parsed.Args) != len(want) {
		t.Fatalf("args len = %d, want %d: %#v", len(parsed.Args), len(want), parsed.Args)
	}
	for i := range want {
		if parsed.Args[i] != want[i] {
			t.Fatalf("arg[%d] = %q, want %q", i, parsed.Args[i], want[i])
		}
	}
}

func TestParseStrategyReportsMissingWinWS(t *testing.T) {
	t.Parallel()
	root := newTestRoot(t)
	strategyPath := filepath.Join(root, "empty.bat")
	if err := os.WriteFile(strategyPath, []byte("@echo off\r\n"), 0600); err != nil {
		t.Fatal(err)
	}

	ports := strategy.GameFilterPorts{}
	if _, err := strategy.Parse(strategy.Strategy{Name: "empty.bat", Path: strategyPath}, root, ports); err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
}

func TestListClassifiesCustomFromConfigAndExtraDirOnly(t *testing.T) {
	t.Parallel()
	root := newTestRoot(t)
	extraDir := filepath.Join(t.TempDir(), "custom_strategies")
	if err := os.MkdirAll(extraDir, 0700); err != nil {
		t.Fatal(err)
	}
	rootBats := []string{
		"general.bat",
		"general (ALT7.1) (custom).bat",
		"general (ALT7.2) (custom).bat",
	}
	for _, name := range rootBats {
		if err := os.WriteFile(filepath.Join(root, name), []byte("winws.exe"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(extraDir, "mine.bat"), []byte("winws.exe"), 0600); err != nil {
		t.Fatal(err)
	}

	customNames := []string{"general (ALT7.2) (custom).bat"}

	got, err := strategy.List(root, customNames, extraDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	wantCustom := map[string]bool{
		"general.bat":                   false,
		"general (ALT7.1) (custom).bat": false,
		"general (ALT7.2) (custom).bat": true,
		"mine.bat":                      true,
	}
	if len(got) != len(wantCustom) {
		t.Fatalf("List() returned %d strategies, want %d: %#v", len(got), len(wantCustom), got)
	}
	for _, s := range got {
		want, ok := wantCustom[s.Name]
		if !ok {
			t.Fatalf("unexpected strategy %q", s.Name)
		}
		if s.Custom != want {
			t.Errorf("%q Custom = %v, want %v", s.Name, s.Custom, want)
		}
	}
}

func newTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"bin", "lists", "utils"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
