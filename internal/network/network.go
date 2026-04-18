package network

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"QMLauncher/internal/version"
	env "QMLauncher/pkg"
)

// QMServerUserAgent is the User-Agent for all QMServer API requests
var QMServerUserAgent = "QMLauncher/" + version.Current

// qmserverTransport adds User-Agent to requests
type qmserverTransport struct {
	rt http.RoundTripper
}

func (t *qmserverTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	if req2.Header == nil {
		req2.Header = make(http.Header)
	}
	for k, v := range req.Header {
		req2.Header[k] = v
	}
	req2.Header.Set("User-Agent", QMServerUserAgent)
	return t.rt.RoundTrip(req2)
}

var qmserverBaseHTTPTransport http.RoundTripper = &http.Transport{
	Proxy:               http.ProxyFromEnvironment,
	TLSHandshakeTimeout: 30 * time.Second,
}

// QMServerHTTPClient is the HTTP client for QMServer API (with proper User-Agent).
// When Debug mode is enabled in launcher settings, requests/responses are traced to *_debug.log.
var QMServerHTTPClient = &http.Client{
	Timeout: 45 * time.Second,
	Transport: &debugCondTransport{
		inner: &qmserverTransport{rt: qmserverBaseHTTPTransport},
	},
}

var externalHTTPTransport http.RoundTripper = &http.Transport{
	Proxy:               http.ProxyFromEnvironment,
	TLSHandshakeTimeout: 30 * time.Second,
}

// HTTPClientForExternal returns an HTTP client (CurseForge, Mojang, etc.) with the same optional debug tracing as QMServerHTTPClient.
func HTTPClientForExternal(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &debugCondTransport{
			inner: externalHTTPTransport,
		},
	}
}

// HTTPClientMetadata is a shared client for small GETs (Forge/Maven/Fabric metadata, Mojang helpers). Debug-traced when launcher_debug is on.
var HTTPClientMetadata = HTTPClientForExternal(120 * time.Second)

// HTTPClientLongDownload is for large files (assets, libraries, updater); no overall client deadline. Debug-traced when launcher_debug is on.
var HTTPClientLongDownload = HTTPClientForExternal(0)

const MaxConcurrentDownloads = 6

type DownloadEntry struct {
	URL      string
	Path     string
	Sha1     string
	FileMode os.FileMode
}

// DownloadFile downloads the specified DownloadEntry and saves it.
//
// All parent directories are created in order to create the file.
func DownloadFile(entry DownloadEntry) error {
	req, err := http.NewRequest(http.MethodGet, entry.URL, nil)
	if err != nil {
		return err
	}
	resp, err := HTTPClientLongDownload.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := CheckResponse(resp); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(entry.Path), 0755); err != nil {
		return fmt.Errorf("create directory for file %q: %w", entry.Path, err)
	}
	out, err := os.Create(entry.Path)
	if err != nil {
		return fmt.Errorf("create file %q: %w", entry.Path, err)
	}
	defer out.Close()

	if entry.FileMode != 0 {
		if err := out.Chmod(entry.FileMode); err != nil {
			return fmt.Errorf("set permissions for file %q: %w", entry.Path, err)
		}
	}

	hash := sha1.New()
	tee := io.TeeReader(resp.Body, hash)

	if _, err := io.Copy(out, tee); err != nil {
		return err
	}

	if entry.Sha1 != "" {
		if hex.EncodeToString(hash.Sum(nil)) != entry.Sha1 {
			return fmt.Errorf("invalid checksum from %q", entry.URL)
		}
	}

	return nil
}

// StartDownloadEntries runs DownloadFile on each specified DownloadEntry and returns a channel with the download results.
func StartDownloadEntries(entries []DownloadEntry) chan error {
	var wg sync.WaitGroup
	results := make(chan error)
	d := make(chan struct{}, MaxConcurrentDownloads)
	for _, entry := range entries {
		wg.Add(1)
		go func(entry DownloadEntry) {
			defer wg.Done()

			d <- struct{}{}
			err := DownloadFile(entry)
			<-d
			results <- err
		}(entry)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	return results
}

// HTTPStatusError is an error type returned when an HTTP response finishes with a status code >= 300 or < 200
type HTTPStatusError struct {
	URL        string
	Method     string
	StatusCode int
}

type qmserverErrorBody struct {
	Detail string `json:"detail"`
	Error  string `json:"error"`
}

// ReadQMServerError reads a concise error message from QMServer response body.
func ReadQMServerError(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	var body qmserverErrorBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return ""
	}
	msg := strings.TrimSpace(body.Detail)
	if msg == "" {
		msg = strings.TrimSpace(body.Error)
	}
	return msg
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("%s %s (%d)", e.Method, e.URL, e.StatusCode)
}

// CheckResponse ensures the status code of an HTTP response is successful, returning an HTTPStatusError if not.
func CheckResponse(resp *http.Response) error {
	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return &HTTPStatusError{
			URL:        resp.Request.URL.String(),
			Method:     resp.Request.Method,
			StatusCode: resp.StatusCode,
		}
	}
	return nil
}

// QMServerInfo represents server information from QMServer
type QMServerInfo struct {
	ID               uint   `json:"id"`
	UUID             string `json:"uuid"`
	Name             string `json:"name"`
	GameKind         string `json:"game_kind"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	Version          string `json:"version"`
	ModLoader        string `json:"mod_loader"`
	ModLoaderVersion string `json:"mod_loader_version"`
	IsPremium        bool   `json:"is_premium"`
	Enabled          *bool  `json:"enabled,omitempty"`
	Players          *int   `json:"players,omitempty"`
	MaxPlayers       *int   `json:"max_players,omitempty"`
	GameServerOnline *bool  `json:"game_server_online,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// ErrServerProfileDisabled is returned when a game server profile exists but is turned off in QMAdmin.
var ErrServerProfileDisabled = errors.New("server profile disabled")

// QMServerProfileEnabled is true when the API marks the profile as enabled (default true if field omitted).
func QMServerProfileEnabled(s QMServerInfo) bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// CheckServerProfileConnectAllowed returns ErrServerProfileDisabled when the cached /servers list
// contains this id and enabled is false. Returns nil when id is 0, list unavailable, or profile is enabled/unknown.
func CheckServerProfileConnectAllowed(serverID uint) error {
	if serverID == 0 {
		return nil
	}
	resp, err := GetQMServersList()
	if err != nil {
		return nil
	}
	for i := range resp.ServerProfiles {
		s := &resp.ServerProfiles[i]
		if s.ID != serverID {
			continue
		}
		if !QMServerProfileEnabled(*s) {
			return ErrServerProfileDisabled
		}
		return nil
	}
	return nil
}

// QMServersResponse represents the response from QMServer servers endpoint
type QMServersResponse struct {
	Count          int            `json:"count"`
	ServerProfiles []QMServerInfo `json:"server_profiles"`
	Error          string         `json:"error,omitempty"`
}

// DefaultQMServerAPIBase returns the QMServer Cloud API base URL (when "use cloud" is on in launcher settings).
// Override via QMSERVER_API_BASE env (e.g. https://api.qx-dev.ru/api/v1).
func DefaultQMServerAPIBase() string {
	if base := strings.TrimSpace(os.Getenv("QMSERVER_API_BASE")); base != "" {
		return strings.TrimSuffix(base, "/")
	}
	return "https://api.qx-dev.ru/api/v1"
}

var (
	apiTargetMu       sync.RWMutex
	useQMServerCloud  = true
	customQMServerAPI = ""
)

// ApplyLauncherAPITarget sets whether the launcher uses the cloud API URL or a custom base (both without trailing slash).
// Persists are handled by the app; this only updates in-memory state used by EffectiveQMServerAPIBase.
func ApplyLauncherAPITarget(useCloud bool, customBase string) {
	customBase = strings.TrimSpace(customBase)
	customBase = strings.TrimSuffix(customBase, "/")
	apiTargetMu.Lock()
	useQMServerCloud = useCloud
	customQMServerAPI = customBase
	apiTargetMu.Unlock()
	InvalidateServersCache()
}

// EffectiveQMServerAPIBase is the API base the launcher must use for /servers, /settings/..., etc.
func EffectiveQMServerAPIBase() string {
	apiTargetMu.RLock()
	defer apiTargetMu.RUnlock()
	if useQMServerCloud {
		return DefaultQMServerAPIBase()
	}
	if customQMServerAPI != "" {
		return customQMServerAPI
	}
	return DefaultQMServerAPIBase()
}

// QMServerBaseURL returns the base URL for a QMServer host:port (uses https for port 443)
func QMServerBaseURL(host string, port int) string {
	scheme := "http"
	if port == 443 {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

const (
	serversCacheTTL   = 5 * time.Minute    // In-memory cache TTL
	serversDiskMaxAge = 7 * 24 * time.Hour // Max age of disk cache when API fails (7 days)
	serversCacheFile  = "servers_cache.json"
)

var (
	serversCacheMu   sync.RWMutex
	serversCache     *QMServersResponse
	serversCacheTime time.Time
)

func keepMinecraftLauncherServers(resp *QMServersResponse) {
	if resp == nil {
		return
	}
	n := 0
	for _, s := range resp.ServerProfiles {
		gk := strings.ToLower(strings.TrimSpace(s.GameKind))
		if gk == "" || gk == "minecraft" {
			resp.ServerProfiles[n] = s
			n++
		}
	}
	resp.ServerProfiles = resp.ServerProfiles[:n]
	resp.Count = n
}

// GetQMServersList fetches the list of servers from QMServer Cloud API.
// Uses in-memory cache (5 min TTL) and disk cache as fallback when API is unavailable.
func GetQMServersList() (*QMServersResponse, error) {
	base := EffectiveQMServerAPIBase()
	url := base + "/servers"

	// 1. Check in-memory cache
	serversCacheMu.RLock()
	if serversCache != nil && time.Since(serversCacheTime) < serversCacheTTL {
		cached := *serversCache
		cached.ServerProfiles = append([]QMServerInfo(nil), serversCache.ServerProfiles...)
		serversCacheMu.RUnlock()
		keepMinecraftLauncherServers(&cached)
		return &cached, nil
	}
	serversCacheMu.RUnlock()

	// 2. Fetch from API (with retries for transient errors: unexpected EOF, connection reset)
	const maxRetries = 3
	var resp *http.Response
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		resp, err = QMServerHTTPClient.Get(url)
		if err == nil {
			break
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "unexpected EOF") &&
			!strings.Contains(errStr, "connection reset") {
			break
		}
	}
	if err != nil {
		if disk := loadServersFromDisk(); disk != nil {
			keepMinecraftLauncherServers(disk)
			return disk, nil
		}
		return nil, fmt.Errorf("failed to connect to QMServer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if disk := loadServersFromDisk(); disk != nil {
			keepMinecraftLauncherServers(disk)
			return disk, nil
		}
		msg := ReadQMServerError(resp)
		if msg != "" {
			return nil, fmt.Errorf("QMServer does not serve QMLauncher: %s", msg)
		}
		return nil, fmt.Errorf("QMServer returned status %d", resp.StatusCode)
	}

	var serversResponse QMServersResponse
	if err := json.NewDecoder(resp.Body).Decode(&serversResponse); err != nil {
		if disk := loadServersFromDisk(); disk != nil {
			keepMinecraftLauncherServers(disk)
			return disk, nil
		}
		return nil, fmt.Errorf("failed to parse servers list: %w", err)
	}

	keepMinecraftLauncherServers(&serversResponse)

	// 4. Update caches
	serversCacheMu.Lock()
	serversCache = &serversResponse
	serversCacheTime = time.Now()
	serversCacheMu.Unlock()
	saveServersToDisk(&serversResponse)

	return &serversResponse, nil
}

func serversCachePath() string {
	return filepath.Join(env.RootDir, serversCacheFile)
}

func loadServersFromDisk() *QMServersResponse {
	path := serversCachePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cached struct {
		Data    QMServersResponse `json:"data"`
		SavedAt int64             `json:"saved_at"`
	}
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil
	}
	if time.Since(time.UnixMilli(cached.SavedAt)) > serversDiskMaxAge {
		return nil
	}
	return &cached.Data
}

func saveServersToDisk(resp *QMServersResponse) {
	if resp == nil {
		return
	}
	path := serversCachePath()
	data, err := json.Marshal(map[string]interface{}{
		"data":     resp,
		"saved_at": time.Now().UnixMilli(),
	})
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, data, 0644)
}

// InvalidateServersCache clears the servers cache (e.g. after manual refresh).
func InvalidateServersCache() {
	serversCacheMu.Lock()
	serversCache = nil
	serversCacheMu.Unlock()
}

// MSAServerSettings is the QMServer public response for Microsoft auth in the launcher.
type MSAServerSettings struct {
	ClientID       string
	AllowMicrosoft bool // false when admin set msa_enabled to false
	FetchOK        bool // HTTP 200 and body parsed
}

// FetchMSAServerSettings reads /settings/qmlauncher-msa-client-id from QMServer.
func FetchMSAServerSettings() MSAServerSettings {
	base := EffectiveQMServerAPIBase()
	url := base + "/settings/qmlauncher-msa-client-id"
	resp, err := QMServerHTTPClient.Get(url)
	if err != nil {
		return MSAServerSettings{AllowMicrosoft: true, FetchOK: false}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return MSAServerSettings{AllowMicrosoft: true, FetchOK: false}
	}
	var data struct {
		MSAClientID string `json:"msa_client_id"`
		MSAEnabled  *bool  `json:"msa_enabled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return MSAServerSettings{AllowMicrosoft: true, FetchOK: false}
	}
	allow := true
	if data.MSAEnabled != nil {
		allow = *data.MSAEnabled
	}
	return MSAServerSettings{
		ClientID:       strings.TrimSpace(data.MSAClientID),
		AllowMicrosoft: allow,
		FetchOK:        true,
	}
}

// IsMSAAuthDisabledByQMServer is true when QMServer explicitly disabled Microsoft login (msa_enabled false).
func IsMSAAuthDisabledByQMServer() bool {
	s := FetchMSAServerSettings()
	return s.FetchOK && !s.AllowMicrosoft
}

// DefaultEmbeddedMSAClientID is the built-in Azure AD (MSA) application ID for QMLauncher.
// New app registrations must be approved for the Minecraft API or login_with_xbox returns 403 — see https://aka.ms/MinecraftAppReg
// QMServer/QMAdmin may override via msa_client_id; set msa_enabled false to block Microsoft login.
const DefaultEmbeddedMSAClientID = "f11a1edf-0b1a-43c0-a249-ecb10a58a797"

// GetQMLauncherMSAClientID fetches the MSA Client ID from QMServer, then env, then the embedded default.
// Returns empty only if Microsoft auth is turned off in QMAdmin/QMServer (successful fetch with msa_enabled false).
func GetQMLauncherMSAClientID() string {
	s := FetchMSAServerSettings()
	if s.FetchOK && !s.AllowMicrosoft {
		return ""
	}
	if s.FetchOK && s.ClientID != "" {
		return s.ClientID
	}
	if v := strings.TrimSpace(os.Getenv("QMLAUNCHER_MSA_CLIENT_ID")); v != "" {
		return v
	}
	return DefaultEmbeddedMSAClientID
}

// FetchLauncherCatalogModules reports which remote store catalogs QMServer allows for a logged-in launcher token.
// The desktop launcher no longer uses this to gate CurseForge/Modrinth; kept for API compatibility.
func FetchLauncherCatalogModules(apiBase, token string) (curseforge, modrinth bool, err error) {
	apiBase = strings.TrimSuffix(strings.TrimSpace(apiBase), "/")
	token = strings.TrimSpace(token)
	if apiBase == "" || token == "" {
		return false, false, errors.New("missing api base or token")
	}
	reqURL := apiBase + "/launcher/catalog-modules?token=" + url.QueryEscape(token)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return false, false, err
	}
	req.Header.Set("User-Agent", QMServerUserAgent)
	resp, err := QMServerHTTPClient.Do(req)
	if err != nil {
		return false, false, err
	}
	defer resp.Body.Close()
	if err := CheckResponse(resp); err != nil {
		return false, false, err
	}
	var body struct {
		Curseforge bool `json:"curseforge"`
		Modrinth   bool `json:"modrinth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, false, err
	}
	return body.Curseforge, body.Modrinth, nil
}
