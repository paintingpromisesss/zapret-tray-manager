package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"zapret-tray-manager/internal/client"
)

const (
	latestReleaseAPI = "https://api.github.com/repos/paintingpromisesss/zapret-tray-manager/releases/latest"
	releasesPageURL  = "https://github.com/paintingpromisesss/zapret-tray-manager/releases/latest"
)

var (
	ErrNoInstallerAsset = errors.New("release has no installer (.exe) asset")
	ErrInvalidVersion   = errors.New("invalid version string")
)

type Release struct {
	Version    string
	TagName    string
	ReleaseURL string
	AssetName  string
	AssetURL   string
}

type githubRelease struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type Client struct {
	http *client.Client
}

func NewClient(http *client.Client) *Client {
	return &Client{http: http}
}

func (c *Client) LatestRelease(parentCtx context.Context) (Release, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	var gh githubRelease
	if err := c.http.GetJSON(ctx, latestReleaseAPI, &gh); err != nil {
		return Release{}, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	return releaseFromGitHub(gh), nil
}

func (c *Client) DownloadInstaller(parentCtx context.Context, release Release) (string, error) {
	if release.AssetURL == "" {
		return "", fmt.Errorf("%w: %s", ErrNoInstallerAsset, release.Version)
	}

	downloadDir := filepath.Join(os.TempDir(), "zapret-tray-update")
	if err := os.MkdirAll(downloadDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create download directory: %w", err)
	}

	assetName := release.AssetName
	if assetName == "" {
		assetName = "zapret-tray-manager-" + release.Version + "-setup.exe"
	}

	ctx, cancel := context.WithTimeout(parentCtx, 120*time.Second)
	defer cancel()

	path, err := c.http.Download(ctx, release.AssetURL, filepath.Join(downloadDir, assetName))
	if err != nil {
		return "", fmt.Errorf("failed to download installer: %w", err)
	}
	return path, nil
}

func releaseFromGitHub(r githubRelease) Release {
	release := Release{
		Version:    NormalizeVersion(r.TagName),
		TagName:    r.TagName,
		ReleaseURL: r.HTMLURL,
	}
	for _, asset := range r.Assets {
		name := strings.ToLower(asset.Name)
		if strings.HasSuffix(name, ".exe") && strings.Contains(name, "setup") {
			release.AssetName = asset.Name
			release.AssetURL = asset.BrowserDownloadURL
			break
		}
	}
	// Fallback: any .exe asset if no "setup"-named one found.
	if release.AssetURL == "" {
		for _, asset := range r.Assets {
			if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") {
				release.AssetName = asset.Name
				release.AssetURL = asset.BrowserDownloadURL
				break
			}
		}
	}
	if release.ReleaseURL == "" {
		release.ReleaseURL = releasesPageURL
	}
	return release
}

func NormalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	return version
}

func IsNewer(current, latest string) bool {
	return CompareVersions(current, latest) < 0
}

func CompareVersions(a, b string) int {
	va, aOK := parseSemver(a)
	vb, bOK := parseSemver(b)

	switch {
	case !aOK && !bOK:
		return 0
	case !aOK:
		return -1
	case !bOK:
		return 1
	}
	return va.compare(vb)
}

type semver struct {
	core       [3]int
	prerelease string
}

func (v semver) compare(other semver) int {
	for i := range v.core {
		if v.core[i] != other.core[i] {
			if v.core[i] < other.core[i] {
				return -1
			}
			return 1
		}
	}
	return comparePrerelease(v.prerelease, other.prerelease)
}

func comparePrerelease(a, b string) int {
	switch {
	case a == "" && b == "":
		return 0
	case a == "":
		return 1
	case b == "":
		return -1
	default:
		return strings.Compare(a, b)
	}
}

func parseSemver(v string) (semver, bool) {
	v = NormalizeVersion(v)
	if v == "" {
		return semver{}, false
	}

	v, _, _ = strings.Cut(v, "+")

	var out semver
	core, prerelease, _ := strings.Cut(v, "-")
	out.prerelease = prerelease

	parts := strings.Split(core, ".")
	for i := range out.core {
		if i >= len(parts) {
			break
		}
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return semver{}, false
		}
		out.core[i] = n
	}
	return out, true
}
