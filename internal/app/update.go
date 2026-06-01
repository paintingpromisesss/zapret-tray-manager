package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"zapret-tray-manager/internal/selfupdate"
)

type UpdateCheck struct {
	Available bool
	Current   string
	Release   selfupdate.Release
}

//nolint:wrapcheck // LatestRelease already wraps with context.
func (a *App) CheckForUpdate(ctx context.Context) (UpdateCheck, error) {
	release, err := a.updateClient.LatestRelease(ctx)
	if err != nil {
		return UpdateCheck{}, err
	}
	return UpdateCheck{
		Available: selfupdate.IsNewer(a.version, release.Version),
		Current:   selfupdate.NormalizeVersion(a.version),
		Release:   release,
	}, nil
}

func (a *App) DownloadAndRunUpdate(ctx context.Context, release selfupdate.Release) error {
	path, err := a.updateClient.DownloadInstaller(ctx, release)
	if err != nil {
		return fmt.Errorf("download update installer: %w", err)
	}
	a.logger.Info("update installer downloaded", "version", release.Version, "path", path)

	//nolint:gosec,noctx // path is our own temp download; installer runs independently of app lifetime.
	cmd := exec.Command(path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch update installer: %w", err)
	}
	a.logger.Info("update installer launched", "pid", cmd.Process.Pid)
	return nil
}
