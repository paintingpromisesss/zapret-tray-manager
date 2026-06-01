//nolint:testpackage // Tests cover unexported releaseFromGitHub / githubRelease parsing.
package selfupdate

import "testing"

func TestCompareVersions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.2", "1.2.0", 0},
		{"1.0.0-rc1", "1.0.0", -1},
		{"1.0.0", "1.0.0-rc1", 1},
		{"1.0.0-rc1", "1.0.0-rc2", -1},
		{"1.0.0+build1", "1.0.0+build2", 0},
		{"dev", "1.0.0", -1},
		{"dev", "dev", 0},
		{"1.0.0", "dev", 1},
	}
	for _, c := range cases {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	t.Parallel()
	if !IsNewer("1.0.0", "1.0.1") {
		t.Error("1.0.1 should be newer than 1.0.0")
	}
	if IsNewer("1.0.1", "1.0.0") {
		t.Error("1.0.0 should not be newer than 1.0.1")
	}
	if IsNewer("1.0.0", "1.0.0") {
		t.Error("equal versions: not newer")
	}
	if IsNewer("dev", "1.0.0") {
		t.Error("dev build treated as latest: no update offered")
	}
}

func TestReleaseFromGitHub(t *testing.T) {
	t.Parallel()
	gh := githubRelease{
		TagName: "v1.2.3",
		HTMLURL: "https://example/r",
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{Name: "notes.txt", BrowserDownloadURL: "https://x/notes.txt"},
			{Name: "Zapret Tray Manager-1.2.3-setup.exe", BrowserDownloadURL: "https://x/setup.exe"},
		},
	}
	r := releaseFromGitHub(gh)
	if r.Version != "1.2.3" {
		t.Errorf("version = %q", r.Version)
	}
	if r.AssetURL != "https://x/setup.exe" {
		t.Errorf("assetURL = %q", r.AssetURL)
	}
}
