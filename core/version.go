package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AppVersion is the current application version.
const AppVersion = "0.6"

// GitHub repository for version checks
const githubRepo = "mimyosa/dmailsender"
const githubAPIURL = "https://api.github.com/repos/" + githubRepo + "/releases/latest"

// VersionCheckResult holds the result of a version check.
type VersionCheckResult struct {
	Current     string `json:"current"`
	Latest      string `json:"latest"`
	UpdateAvail bool   `json:"update_avail"`
	DownloadURL string `json:"download_url,omitempty"`
	Error       string `json:"error,omitempty"`
}

// githubRelease is a minimal struct for GitHub Releases API response.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckVersion fetches the latest version from GitHub Releases and compares.
func CheckVersion() VersionCheckResult {
	result := VersionCheckResult{Current: "v" + AppVersion}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		result.Error = "Version check failed"
		return result
	}
	req.Header.Set("User-Agent", "dMailSender/"+AppVersion)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		result.Error = "Version check failed: network error"
		return result
	}
	defer resp.Body.Close()

	// 404 means no releases yet — treat as up to date
	if resp.StatusCode == http.StatusNotFound {
		result.Latest = result.Current
		return result
	}

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("Version check failed: HTTP %d", resp.StatusCode)
		return result
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = "Version check failed: read error"
		return result
	}

	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		result.Error = "Version check failed: parse error"
		return result
	}

	latest := strings.TrimSpace(release.TagName)
	if latest == "" {
		result.Latest = result.Current
		return result
	}

	// Normalize: ensure "v" prefix
	if !strings.HasPrefix(latest, "v") {
		latest = "v" + latest
	}
	result.Latest = latest

	// Compare versions: strip "v" prefix and compare
	currentVer := strings.TrimPrefix(result.Current, "v")
	latestVer := strings.TrimPrefix(latest, "v")

	if compareVersions(latestVer, currentVer) > 0 {
		result.UpdateAvail = true
		result.DownloadURL = release.HTMLURL
	}

	return result
}

// compareVersions compares two dot-separated version strings.
// Returns >0 if a > b, <0 if a < b, 0 if equal.
func compareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		var aNum, bNum int
		if i < len(aParts) {
			fmt.Sscanf(aParts[i], "%d", &aNum)
		}
		if i < len(bParts) {
			fmt.Sscanf(bParts[i], "%d", &bNum)
		}
		if aNum != bNum {
			return aNum - bNum
		}
	}
	return 0
}
