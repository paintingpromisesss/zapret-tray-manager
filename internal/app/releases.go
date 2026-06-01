package app

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zapret-tray-manager/internal/config"
	"zapret-tray-manager/internal/manager"
	"zapret-tray-manager/internal/zapretver"
)

var (
	ErrInvalidLocalZapretFolder = errors.New("local zapret version folder is invalid")
	ErrZipAssetNotFound         = errors.New("zip asset not found in zapret release")
	ErrUnsafeZipPath            = errors.New("unsafe path in zip archive")
	ErrZipEntryTooLarge         = errors.New("zip entry exceeds maximum allowed size")
)

const maxZipEntrySize = 500 << 20

//nolint:wrapcheck // No need to wrap errors that already have context or are self-explanatory.
func (a *App) FetchZapretReleases(ctx context.Context, limit int) ([]zapretver.Release, error) {
	return a.verClient.FetchReleases(ctx, limit)
}

//nolint:wrapcheck // No need to wrap errors that already have context or are self-explanatory.
func (a *App) DownloadZapretRelease(ctx context.Context, release zapretver.Release) (string, error) {
	return a.verClient.DownloadArchive(ctx, release)
}

func (a *App) LocalZapretRoots() []LocalZapretRoot {
	programDir := config.ExecutableDir()
	entries, err := os.ReadDir(programDir)
	if err != nil {
		a.logger.Warn("failed to read program dir", "dir", programDir, "error", err)
		return nil
	}

	var roots []LocalZapretRoot
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(strings.ToLower(entry.Name()), "zapret-") {
			continue
		}

		base := filepath.Join(programDir, entry.Name())
		root, err := a.FindZapretRoot(base)
		if err != nil {
			continue
		}
		roots = append(roots, LocalZapretRoot{
			Version: zapretver.NormalizeVersion(strings.TrimPrefix(entry.Name(), "zapret-")),
			Path:    root,
		})
	}
	return roots
}

func (a *App) ReleaseRootPath(release zapretver.Release) string {
	return filepath.Join(config.ExecutableDir(), "zapret-"+zapretver.NormalizeVersion(release.Version))
}

func (a *App) IsReleaseDownloaded(release zapretver.Release) bool {
	_, err := a.FindZapretRoot(a.ReleaseRootPath(release))
	return err == nil
}

func (a *App) IsExeLocalRoot(root string) bool {
	execDir := config.ExecutableDir()
	rel, err := filepath.Rel(execDir, filepath.Clean(root))
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") && strings.HasPrefix(strings.ToLower(rel), "zapret-")
}

func (a *App) SwitchToLocalRoot(root string) error {
	status, err := a.manager.ServiceStatus()
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}
	if err = a.SetRoot(root); err != nil {
		return err
	}
	if !a.IsExeLocalRoot(root) {
		err = a.AddUserLocalRoot(root)
		if err != nil {
			return fmt.Errorf("failed to add user local root: %w", err)
		}
	}
	return a.reconfigureAfterRootSwitch(status)
}

func (a *App) SwitchToZapretRelease(ctx context.Context, release zapretver.Release) error {
	root, downloaded, err := a.ensureZapretRelease(ctx, release)
	if err != nil {
		return err
	}
	if downloaded {
		a.logger.Info("zapret release downloaded and extracted", "version", release.Version, "root", root)
	}
	//nolint:contextcheck // SetRoot's global-settings apply uses its own bounded context by design.
	return a.SwitchToLocalRoot(root)
}

func (a *App) ensureZapretRelease(ctx context.Context, release zapretver.Release) (string, bool, error) {
	targetDir := a.ReleaseRootPath(release)
	if root, err := a.FindZapretRoot(targetDir); err == nil {
		return root, false, nil
	}
	if _, err := os.Stat(targetDir); err == nil {
		return "", false, fmt.Errorf("%w: %s", ErrInvalidLocalZapretFolder, targetDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, fmt.Errorf("failed to check target dir: %w", err)
	}
	if release.AssetURL == "" {
		return "", false, fmt.Errorf("%w: %s", ErrZipAssetNotFound, release.Version)
	}

	archivePath, err := a.DownloadZapretRelease(ctx, release)
	if err != nil {
		return "", false, err
	}

	tmpDir := targetDir + ".tmp-" + time.Now().Format("20060102150405")
	if err = os.RemoveAll(tmpDir); err != nil {
		return "", false, fmt.Errorf("failed to remove existing temp dir: %w", err)
	}
	if err = os.MkdirAll(tmpDir, 0700); err != nil {
		return "", false, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err = extractZipArchive(archivePath, tmpDir); err != nil {
		return "", false, fmt.Errorf("failed to extract zip archive: %w", err)
	}
	extractedRoot, err := a.FindZapretRoot(tmpDir)
	if err != nil {
		return "", false, fmt.Errorf("failed to find zapret root: %w", err)
	}

	relativeRoot, err := filepath.Rel(tmpDir, extractedRoot)
	if err != nil {
		return "", false, fmt.Errorf("failed to calculate relative path: %w", err)
	}
	if err := os.Rename(tmpDir, targetDir); err != nil {
		return "", false, fmt.Errorf("failed to rename temp dir: %w", err)
	}
	return filepath.Join(targetDir, relativeRoot), true, nil
}

func (a *App) FindZapretRoot(base string) (string, error) {
	if strings.TrimSpace(base) == "" {
		return "", ErrZapretRootNotFound
	}

	var found string
	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			//nolint:nilerr // Skip unreadable entries and keep walking the tree.
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if rel, err := filepath.Rel(base, path); err == nil && rel != "." && strings.Count(rel, string(os.PathSeparator)) > 2 {
			return filepath.SkipDir
		}

		if err := os.MkdirAll(filepath.Join(path, "utils"), 0700); err != nil {
			//nolint:nilerr // Cannot prepare this candidate dir; skip it and keep walking.
			return nil
		}
		candidate := manager.New(path, a.logger)
		if candidate.Validate() == nil {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to walk directory: %w", err)
	}
	if found == "" {
		return "", fmt.Errorf("%w: %s", ErrZapretRootNotFound, base)
	}
	return found, nil
}

//nolint:gocyclo,cyclop // Sequential zip-entry handling (dir/file, path safety, copy, close) is clearer inline than split.
func extractZipArchive(archivePath string, targetDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip archive: %w", err)
	}
	defer func() { _ = reader.Close() }()

	targetDir, err = filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	for _, file := range reader.File {
		targetPath, err := safeZipTarget(targetDir, file.Name)
		if err != nil {
			return fmt.Errorf("failed to calculate safe target path: %w", err)
		}
		if file.FileInfo().IsDir() {
			if err = os.MkdirAll(targetPath, 0700); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}
		if err = os.MkdirAll(filepath.Dir(targetPath), 0700); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		source, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip file: %w", err)
		}
		//nolint:gosec // targetPath is constrained to targetDir by safeZipTarget above.
		target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			_ = source.Close()
			return fmt.Errorf("failed to create target file: %w", err)
		}
		written, copyErr := io.Copy(target, io.LimitReader(source, maxZipEntrySize+1))
		closeSourceErr := source.Close()
		closeTargetErr := target.Close()
		if copyErr != nil {
			return fmt.Errorf("failed to copy file: %w", copyErr)
		}
		if written > maxZipEntrySize {
			return fmt.Errorf("%w: %s", ErrZipEntryTooLarge, file.Name)
		}
		if closeSourceErr != nil {
			return fmt.Errorf("failed to close source file: %w", closeSourceErr)
		}
		if closeTargetErr != nil {
			return fmt.Errorf("failed to close target file: %w", closeTargetErr)
		}
	}
	return nil
}

func safeZipTarget(targetDir string, name string) (string, error) {
	cleanName := filepath.Clean(name)
	if cleanName == "." || filepath.IsAbs(cleanName) || strings.HasPrefix(cleanName, ".."+string(os.PathSeparator)) || cleanName == ".." {
		return "", fmt.Errorf("%w: %s", ErrUnsafeZipPath, name)
	}
	targetPath := filepath.Join(targetDir, cleanName)
	rel, err := filepath.Rel(targetDir, targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative path: %w", err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: %s", ErrUnsafeZipPath, name)
	}
	return targetPath, nil
}
