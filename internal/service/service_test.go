package service_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zapret-tray-manager/internal/service"
	"zapret-tray-manager/internal/strategy"
	"zapret-tray-manager/internal/winexec"
)

var (
	ErrServiceDoesNotExist = errors.New("service does not exist")
	ErrSCQueryFailed       = errors.New("sc.exe query zapret: exit status 1: [SC] EnumQueryServicesStatus:OpenService FAILED 1060")
)

func TestParseServiceState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		output string
		want   service.State
	}{
		{
			name: "running",
			output: `SERVICE_NAME: zapret
        STATE              : 4  RUNNING`,
			want: service.StateRunning,
		},
		{
			name:   "stopped",
			output: `STATE              : 1  STOPPED`,
			want:   service.StateStopped,
		},
		{
			name:   "start pending",
			output: `STATE              : 2  START_PENDING`,
			want:   service.StateStartPending,
		},
		{
			name:   "unknown",
			output: `STATE              : 99  PAUSED`,
			want:   service.StateUnknown,
		},
		{
			name:   "no state",
			output: `SERVICE_NAME: zapret`,
			want:   service.StateUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runner := &fakeRunner{
				outs: map[string]string{
					"sc.exe query zapret":                     tt.output,
					"tasklist.exe /FI IMAGENAME eq winws.exe": "INFO: No tasks are running which match the specified criteria.",
					`reg.exe query HKLM\System\CurrentControlSet\Services\zapret /v zapret-discord-youtube`: "zapret-discord-youtube    REG_SZ    general",
				},
				errs: map[string]error{
					"sc.exe query WinDivert": ErrServiceDoesNotExist,
				},
			}
			svc := service.New("", nil, service.WithRunner(runner))
			status, _ := svc.Status()
			if status.Zapret != tt.want {
				t.Fatalf("Status().Zapret = %q, want %q", status.Zapret, tt.want)
			}
		})
	}
}

func TestStatusPreservesStrategyNameWithSpaces(t *testing.T) {
	t.Parallel()
	const wantStrategy = "general (ALT7.1)  (custom)"
	runner := &fakeRunner{
		outs: map[string]string{
			"sc.exe query zapret":                     "STATE              : 4  RUNNING",
			"tasklist.exe /FI IMAGENAME eq winws.exe": "INFO: No tasks are running which match the specified criteria.",
			`reg.exe query HKLM\System\CurrentControlSet\Services\zapret /v zapret-discord-youtube`: "    zapret-discord-youtube    REG_SZ    " + wantStrategy,
		},
		errs: map[string]error{
			"sc.exe query WinDivert": ErrServiceDoesNotExist,
		},
	}
	svc := service.New("", nil, service.WithRunner(runner))
	status, _ := svc.Status()
	if status.InstalledStrategy != wantStrategy {
		t.Fatalf("InstalledStrategy = %q, want %q", status.InstalledStrategy, wantStrategy)
	}
}

func TestConfigureWritesConfigAndCallsCommands(t *testing.T) {
	t.Parallel()
	root := newTestRoot(t)
	strategyPath := filepath.Join(root, "general.bat")
	if err := os.WriteFile(strategyPath, []byte(`start "" "%BIN%winws.exe" --arg="has space" --plain=value`), 0600); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		errs: map[string]error{
			"sc.exe query zapret": ErrSCQueryFailed,
		},
	}
	svc := service.New(root, nil, service.WithRunner(runner))

	ports := service.GameFilterPorts{All: "12", TCP: "12", UDP: "12"}
	err := svc.Configure(strategy.Strategy{Name: "general.bat", Path: strategyPath}, ports, true, false)
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	if !runner.called("netsh.exe interface tcp set global timestamps=enabled") {
		t.Fatal("netsh command was not called")
	}
	if !runner.called("sc.exe create zapret") {
		t.Fatalf("sc create was not called; calls: %#v", runner.calls)
	}
	winwsPath := filepath.Join(root, "bin", "winws.exe")
	if !runner.called(`sc.exe create zapret binPath="` + winwsPath + `"`) {
		t.Fatalf("sc create did not include winws.exe in binPath; calls: %#v", runner.calls)
	}
	if runner.called("sc.exe start zapret") {
		t.Fatal("sc start was called despite start=false")
	}
}

type fakeRunner struct {
	errs  map[string]error
	outs  map[string]string
	calls []string
}

func (r *fakeRunner) Run(name string, args ...string) (winexec.Result, error) {
	call := strings.TrimSpace(name + " " + strings.Join(args, " "))
	r.calls = append(r.calls, call)
	result := winexec.Result{Name: name, Args: args, Output: r.outs[call]}
	if err := r.errs[call]; err != nil {
		return result, err
	}
	return result, nil
}

func (r *fakeRunner) called(prefix string) bool {
	for _, call := range r.calls {
		if strings.HasPrefix(call, prefix) {
			return true
		}
	}
	return false
}

func newTestRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "zapret root")
	for _, dir := range []string{"bin", "lists", "utils"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{
		"service.bat",
		filepath.Join("bin", "winws.exe"),
		filepath.Join("lists", "ipset-all.txt"),
	} {
		if err := os.WriteFile(filepath.Join(root, file), []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
