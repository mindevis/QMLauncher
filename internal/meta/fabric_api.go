package meta

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"QMLauncher/internal/network"
)

// FabricAPIModrinthSlug is the Modrinth project slug for Fabric API (https://modrinth.com/mod/fabric-api).
const FabricAPIModrinthSlug = "fabric-api"

// FabricAPIModrinthProjectID is the Modrinth v2 project id for Fabric API.
const FabricAPIModrinthProjectID = "P7dR8mSH"

type modrinthFabricAPIVersion struct {
	GameVersions []string `json:"game_versions"`
	Loaders      []string `json:"loaders"`
	Files        []struct {
		Hashes struct {
			Sha1 string `json:"sha1"`
		} `json:"hashes"`
		URL      string `json:"url"`
		Filename string `json:"filename"`
		Primary  bool   `json:"primary"`
	} `json:"files"`
}

// DownloadFabricAPIIfNeeded downloads the Fabric API mod from Modrinth into modsDir when no fabric-api jar is present.
// Returns the basename of the saved .jar when a new file was downloaded, or "" if Fabric API was already present.
func DownloadFabricAPIIfNeeded(modsDir, cachesDir, gameVersion string) (string, error) {
	if strings.TrimSpace(gameVersion) == "" {
		return "", fmt.Errorf("game version is empty")
	}
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		return "", fmt.Errorf("create mods directory: %w", err)
	}
	present, err := fabricAPIJarPresent(modsDir)
	if err != nil {
		return "", err
	}
	if present {
		return "", nil
	}

	vers, err := fetchFabricAPIVersionsForMC(cachesDir, gameVersion)
	if err != nil {
		return "", err
	}

	var dlURL, sha1hex, fname string
	for _, v := range vers {
		dlURL, sha1hex, fname = pickFabricAPIJarFile(v)
		if dlURL != "" && fname != "" {
			break
		}
	}
	if dlURL == "" || fname == "" {
		return "", fmt.Errorf("no Fabric API file for Minecraft %s (see https://modrinth.com/mod/%s)", gameVersion, FabricAPIModrinthSlug)
	}

	dest := filepath.Join(modsDir, fname)
	if err := network.DownloadFile(network.DownloadEntry{
		URL:  dlURL,
		Path: dest,
		Sha1: strings.TrimSpace(sha1hex),
	}); err != nil {
		return "", err
	}
	return filepath.Base(fname), nil
}

func fabricAPIJarPresent(modsDir string) (bool, error) {
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := strings.ToLower(e.Name())
		if !strings.HasSuffix(n, ".jar") {
			continue
		}
		if strings.Contains(n, "fabric-api") {
			return true, nil
		}
	}
	return false, nil
}

func pickFabricAPIJarFile(v modrinthFabricAPIVersion) (url, sha1, filename string) {
	for _, f := range v.Files {
		if !f.Primary {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(f.Filename), ".jar") {
			continue
		}
		return f.URL, f.Hashes.Sha1, f.Filename
	}
	for _, f := range v.Files {
		if strings.HasSuffix(strings.ToLower(f.Filename), ".jar") {
			return f.URL, f.Hashes.Sha1, f.Filename
		}
	}
	return "", "", ""
}

func fetchFabricAPIVersionsForMC(cachesDir, gameVersion string) ([]modrinthFabricAPIVersion, error) {
	base := fmt.Sprintf("https://api.modrinth.com/v2/project/%s/version", FabricAPIModrinthSlug)

	// Prefer exact game version + fabric loader (Modrinth filter).
	q := url.Values{}
	q.Set("loaders", `["fabric"]`)
	q.Set("game_versions", fmt.Sprintf(`["%s"]`, gameVersion))

	filteredURL := base + "?" + q.Encode()
	cachePath := filepath.Join(cachesDir, "modrinth", "fabric_api_"+sanitizeVerForCache(gameVersion)+".json")
	cache := network.Cache[[]modrinthFabricAPIVersion]{
		Path:        cachePath,
		URL:         filteredURL,
		AlwaysFetch: false,
	}

	var filtered []modrinthFabricAPIVersion
	if err := cache.Get(&filtered); err == nil && len(filtered) > 0 {
		return filtered, nil
	}

	// Broader list: all recent Fabric API builds for Fabric loader, pick first matching game version.
	q2 := url.Values{}
	q2.Set("loaders", `["fabric"]`)
	q2.Set("limit", "100")
	broadURL := base + "?" + q2.Encode()

	body, err := modrinthGET(broadURL)
	if err != nil {
		return nil, err
	}
	var broad []modrinthFabricAPIVersion
	if err := json.Unmarshal(body, &broad); err != nil {
		return nil, fmt.Errorf("parse Modrinth versions: %w", err)
	}

	var out []modrinthFabricAPIVersion
	for _, v := range broad {
		if !versionListsFabric(v) {
			continue
		}
		if !versionListsGame(v, gameVersion) {
			continue
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no Fabric API on Modrinth for Minecraft %s", gameVersion)
	}
	return out, nil
}

func versionListsFabric(v modrinthFabricAPIVersion) bool {
	for _, l := range v.Loaders {
		if strings.EqualFold(l, "fabric") {
			return true
		}
	}
	return false
}

func versionListsGame(v modrinthFabricAPIVersion, gameVersion string) bool {
	for _, g := range v.GameVersions {
		if g == gameVersion {
			return true
		}
	}
	return false
}

func modrinthGET(urlStr string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("modrinth request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("modrinth HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(slurp)))
	}
	return io.ReadAll(resp.Body)
}

func sanitizeVerForCache(v string) string {
	s := strings.ReplaceAll(v, string(filepath.Separator), "_")
	s = strings.ReplaceAll(s, "..", "_")
	if s == "" {
		return "unknown"
	}
	return s
}
