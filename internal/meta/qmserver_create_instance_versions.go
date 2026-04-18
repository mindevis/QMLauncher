package meta

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"QMLauncher/internal/network"
)

type qmserverLauncherVersionsResponse struct {
	Versions []string `json:"versions"`
	Error    string   `json:"error,omitempty"`
}

// FetchCreateInstanceMinecraftFromQMServerCloud loads release ids from QMServer Cloud
// GET {apiBase}/launcher/create-instance/minecraft-versions
func FetchCreateInstanceMinecraftFromQMServerCloud(apiBase string) ([]string, error) {
	apiBase = strings.TrimSuffix(strings.TrimSpace(apiBase), "/")
	if apiBase == "" {
		return nil, fmt.Errorf("empty QMServer API base")
	}
	reqURL := apiBase + "/launcher/create-instance/minecraft-versions"
	resp, err := network.QMServerHTTPClient.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloud returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var data qmserverLauncherVersionsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse cloud response: %w", err)
	}
	if data.Error != "" {
		return nil, fmt.Errorf("cloud error: %s", data.Error)
	}
	return data.Versions, nil
}

// FetchCreateInstanceLoaderFromQMServerCloud loads loader versions from QMServer Cloud
// GET {apiBase}/launcher/create-instance/loader-versions?loader=&game_version=
func FetchCreateInstanceLoaderFromQMServerCloud(apiBase, loader, gameVersion string) ([]string, error) {
	apiBase = strings.TrimSuffix(strings.TrimSpace(apiBase), "/")
	if apiBase == "" {
		return nil, fmt.Errorf("empty QMServer API base")
	}
	u, err := url.Parse(apiBase + "/launcher/create-instance/loader-versions")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("loader", loader)
	q.Set("game_version", gameVersion)
	u.RawQuery = q.Encode()
	resp, err := network.QMServerHTTPClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloud returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var data qmserverLauncherVersionsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse cloud response: %w", err)
	}
	if data.Error != "" {
		return nil, fmt.Errorf("cloud error: %s", data.Error)
	}
	return data.Versions, nil
}
