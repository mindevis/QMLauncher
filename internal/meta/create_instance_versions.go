package meta

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"QMLauncher/internal/network"

	"golang.org/x/mod/semver"
)

// ListCreateInstanceMinecraftReleases returns Minecraft release version ids in manifest order (newest first).
func ListCreateInstanceMinecraftReleases(cachesDir string) ([]string, error) {
	manifest, err := FetchVersionManifest(cachesDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, v := range manifest.Versions {
		if v.Type == "release" {
			out = append(out, v.ID)
		}
	}
	sortVersionsSemverDesc(out)
	return out, nil
}

const maxCreateInstanceLoaderVersions = 120

type mavenVersionsResponse struct {
	Versions []string `json:"versions"`
}

// ListCreateInstanceLoaderVersions returns installable loader versions for the given mod loader and Minecraft version.
// Order is newest-first where the upstream API lists older-first (Forge / NeoForge).
func ListCreateInstanceLoaderVersions(loader string, gameVersion string) ([]string, error) {
	gameVersion = strings.TrimSpace(gameVersion)
	if gameVersion == "" {
		return nil, fmt.Errorf("game version is empty")
	}
	switch strings.ToLower(strings.TrimSpace(loader)) {
	case "fabric":
		return listFabricLikeLoaderVersionsForGame(Fabric, gameVersion)
	case "quilt":
		return listFabricLikeLoaderVersionsForGame(Quilt, gameVersion)
	case "forge":
		return listForgeInstallerVersions(gameVersion)
	case "neoforge":
		return listNeoForgeLoaderVersions(gameVersion)
	default:
		return nil, nil
	}
}

func listFabricLikeLoaderVersionsForGame(api fabricAPI, gameVersion string) ([]string, error) {
	u := fmt.Sprintf("%s/versions/loader/%s", api.url, url.PathEscape(gameVersion))
	resp, err := network.HTTPClientMetadata.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := network.CheckResponse(resp); err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var entries []struct {
		Loader struct {
			Version string `json:"version"`
		} `json:"loader"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parse %s loader list: %w", api.name, err)
	}
	seen := make(map[string]struct{})
	var out []string
	for _, e := range entries {
		ver := strings.TrimSpace(e.Loader.Version)
		if ver == "" {
			continue
		}
		if _, ok := seen[ver]; ok {
			continue
		}
		seen[ver] = struct{}{}
		out = append(out, ver)
	}
	sortVersionsSemverDesc(out)
	return capLoaderList(out, maxCreateInstanceLoaderVersions), nil
}

func listForgeInstallerVersions(gameVersion string) ([]string, error) {
	filter := url.QueryEscape(gameVersion + "-")
	u := "https://maven.minecraftforge.net/api/maven/versions/releases/net/minecraftforge/forge?filter=" + filter
	vers, err := fetchMavenVersionsList(u)
	if err != nil {
		return nil, err
	}
	sortVersionsSemverDesc(vers)
	return capLoaderList(vers, maxCreateInstanceLoaderVersions), nil
}

func listNeoForgeLoaderVersions(gameVersion string) ([]string, error) {
	var u string
	if gameVersion == "1.20.1" {
		u = "https://maven.neoforged.net/api/maven/versions/releases/net/neoforged/forge?filter=" + url.QueryEscape("1.20.1-")
	} else {
		parts := strings.Split(gameVersion, ".")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid game version")
		}
		filter := url.QueryEscape(strings.Join(parts[1:], "."))
		u = "https://maven.neoforged.net/api/maven/versions/releases/net/neoforged/neoforge?filter=" + filter
	}
	vers, err := fetchMavenVersionsList(u)
	if err != nil {
		return nil, err
	}
	if gameVersion != "1.20.1" {
		vers = filterNeoForgeArtifactVersionsForMC(vers, gameVersion)
	}
	sortVersionsSemverDesc(vers)
	return capLoaderList(vers, maxCreateInstanceLoaderVersions), nil
}

func neoForgeArtifactVersionPrefix(gameVersion string) string {
	parts := strings.Split(strings.TrimSpace(gameVersion), ".")
	if len(parts) >= 3 {
		return parts[1] + "." + parts[2] + "."
	}
	if len(parts) == 2 {
		return parts[1] + "."
	}
	return ""
}

func filterNeoForgeArtifactVersionsForMC(vers []string, gameVersion string) []string {
	prefix := neoForgeArtifactVersionPrefix(gameVersion)
	if prefix == "" {
		return vers
	}
	out := make([]string, 0, len(vers))
	for _, v := range vers {
		if strings.HasPrefix(strings.TrimSpace(v), prefix) {
			out = append(out, v)
		}
	}
	return out
}

func fetchMavenVersionsList(rawURL string) ([]string, error) {
	resp, err := network.HTTPClientMetadata.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := network.CheckResponse(resp); err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data mavenVersionsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse maven versions: %w", err)
	}
	return data.Versions, nil
}

func sortVersionsSemverDesc(versions []string) {
	if len(versions) < 2 {
		return
	}
	sort.SliceStable(versions, func(i, j int) bool {
		return versionLessDesc(versions[i], versions[j])
	})
}

func versionLessDesc(i, j string) bool {
	ci, cj := semverCanonical(i), semverCanonical(j)
	if ci != "" && cj != "" {
		return semver.Compare(ci, cj) > 0
	}
	if ci != "" {
		return true
	}
	if cj != "" {
		return false
	}
	return strings.TrimSpace(i) > strings.TrimSpace(j)
}

func semverCanonical(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	v := s
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return ""
	}
	return semver.Canonical(v)
}

func capLoaderList(s []string, max int) []string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
