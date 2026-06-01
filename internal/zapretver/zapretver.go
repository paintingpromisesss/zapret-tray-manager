package zapretver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zapret-tray-manager/internal/client"
)

const (
	releasesAPI    = "https://api.github.com/repos/Flowseal/zapret-discord-youtube/releases"
	defaultPerPage = 10
)

type Release struct {
	Version    string
	TagName    string
	ReleaseURL string
	AssetName  string
	AssetURL   string
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
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

func (c *Client) FetchReleases(parentCtx context.Context, limit int) ([]Release, error) {
	if limit <= 0 {
		limit = defaultPerPage
	}

	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s?per_page=%d", releasesAPI, limit)
	var ghReleases []githubRelease
	if err := c.http.GetJSON(ctx, url, &ghReleases); err != nil {
		return nil, fmt.Errorf("failed to fetch zapret releases: %w", err)
	}

	releases := make([]Release, 0, len(ghReleases))
	for _, r := range ghReleases {
		releases = append(releases, releaseFromGitHub(r))
	}
	return releases, nil
}

func (c *Client) DownloadArchive(parentCtx context.Context, release Release) (string, error) {
	downloadDir := filepath.Join(os.TempDir(), "zapret-tray")
	if err := os.MkdirAll(downloadDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create download directory: %w", err)
	}

	assetName := release.AssetName
	if assetName == "" {
		assetName = "zapret-" + release.Version + ".zip"
	}

	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()

	downloadPath, err := c.http.Download(ctx, release.AssetURL, filepath.Join(downloadDir, assetName))
	if err != nil {
		return "", fmt.Errorf("failed to download zapret release archive: %w", err)
	}
	return downloadPath, nil
}

func NormalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	return version
}

func releaseFromGitHub(r githubRelease) Release {
	release := Release{
		Version:    NormalizeVersion(r.TagName),
		TagName:    r.TagName,
		ReleaseURL: r.HTMLURL,
	}

	for _, asset := range r.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), ".zip") {
			release.AssetName = asset.Name
			release.AssetURL = asset.BrowserDownloadURL
			break
		}
	}

	if release.ReleaseURL == "" && release.TagName != "" {
		release.ReleaseURL = releasesAPI + "/tag/" + release.TagName
	}
	return release
}
