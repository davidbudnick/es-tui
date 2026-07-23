package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

const githubRepo = "davidbudnick/es-tui"

// Overridable for tests.
var (
	updateHTTPClient = &http.Client{Timeout: 8 * time.Second}
	updateAPIBase    = "https://api.github.com"
	osExecutable     = os.Executable
)

type githubLatestRelease struct {
	TagName string `json:"tag_name"`
}

// CheckForUpdate polls GitHub for a newer release and returns UpdateAvailableMsg.
// No-ops for dev / non-semver builds so local runs stay quiet.
func CheckForUpdate(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		return checkForUpdate(currentVersion)
	}
}

func checkForUpdate(currentVersion string) types.UpdateAvailableMsg {
	currentVersion = strings.TrimSpace(currentVersion)
	if currentVersion == "" || currentVersion == "dev" || currentVersion == "ci" || !looksLikeSemver(currentVersion) {
		return types.UpdateAvailableMsg{}
	}

	latest, err := fetchLatestReleaseTag()
	if err != nil {
		return types.UpdateAvailableMsg{Err: err}
	}
	if !isNewerVersion(latest, currentVersion) {
		return types.UpdateAvailableMsg{}
	}

	return types.UpdateAvailableMsg{
		LatestVersion: latest,
		UpgradeCmd:    upgradeHint(),
	}
}

func fetchLatestReleaseTag() (string, error) {
	url := updateAPIBase + "/repos/" + githubRepo + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "es-tui-update-check")

	resp, err := updateHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}
	var rel githubLatestRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("empty tag name")
	}
	return rel.TagName, nil
}

func looksLikeSemver(v string) bool {
	v = strings.TrimPrefix(v, "v")
	return regexp.MustCompile(`^\d+\.\d+\.\d+`).MatchString(v)
}

// isNewerVersion reports whether latest is strictly newer than current.
func isNewerVersion(latest, current string) bool {
	lv := parseSemver(latest)
	cv := parseSemver(current)
	if lv == nil || cv == nil {
		return strings.TrimPrefix(latest, "v") != strings.TrimPrefix(current, "v")
	}
	for i := 0; i < 3; i++ {
		if lv[i] > cv[i] {
			return true
		}
		if lv[i] < cv[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	// strip pre-release / build metadata
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 3 {
		return nil
	}
	out := make([]int, 3)
	for i := 0; i < 3; i++ {
		n := 0
		for _, ch := range parts[i] {
			if ch < '0' || ch > '9' {
				return nil
			}
			n = n*10 + int(ch-'0')
		}
		out[i] = n
	}
	return out
}

func upgradeHint() string {
	path, err := osExecutable()
	if err != nil {
		return "es-tui --update"
	}
	if resolved, rerr := filepath.EvalSymlinks(path); rerr == nil {
		path = resolved
	}
	if strings.Contains(path, "Homebrew") || strings.Contains(path, "homebrew") || strings.Contains(path, "Cellar") {
		return "brew upgrade es-tui"
	}
	return "es-tui --update"
}
