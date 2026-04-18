package meta

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"QMLauncher/internal/debuglog"
	"QMLauncher/internal/network"
	"QMLauncher/internal/version"
)

const (
	cfwidgetAPIBase = "https://api.cfwidget.com"

	minecraftGameID = 432
	cfClassMods     = 6
	cfClassRP       = 12
	cfClassModpacks = 4471
	// Approximate CurseForge class ids for Minecraft Java; search still works if off.
	cfClassShaders   = 6552
	cfClassDatapacks = 6941
)

// RemoteStoreSide is one store's identity for a catalog row (used when a project exists on both sites).
type RemoteStoreSide struct {
	ProjectID string `json:"projectId"`
	Slug      string `json:"slug"`
	PageURL   string `json:"pageUrl"`
	Downloads int64  `json:"downloads"`
}

// RemoteStoreHit is one search result from CurseForge or Modrinth for the resource store UI.
// When Source is "both", Cf and Mr are set; top-level ProjectID/Slug may be empty.
type RemoteStoreHit struct {
	Source    string           `json:"source"`
	ProjectID string           `json:"projectId"`
	Slug      string           `json:"slug"`
	Title     string           `json:"title"`
	Summary   string           `json:"summary"`
	IconURL   string           `json:"iconUrl"`
	PageURL   string           `json:"pageUrl"`
	Downloads int64            `json:"downloads"`
	Cf        *RemoteStoreSide `json:"cf,omitempty"`
	Mr        *RemoteStoreSide `json:"mr,omitempty"`
}

// cfwidgetProjectResp is a subset of api.cfwidget.com JSON for /minecraft/{segment}/{slug}.
type cfwidgetProjectResp struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Thumbnail string `json:"thumbnail"`
	Downloads struct {
		Total int64 `json:"total"`
	} `json:"downloads"`
	URLs struct {
		Curseforge string `json:"curseforge"`
	} `json:"urls"`
}

type curseForgeSearchAPIResponse struct {
	Data []struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		Slug          string `json:"slug"`
		Summary       string `json:"summary"`
		DownloadCount int64  `json:"downloadCount"`
		Links         struct {
			WebsiteURL string `json:"websiteUrl"`
		} `json:"links"`
		Logo struct {
			ThumbnailURL string `json:"thumbnailUrl"`
			URL          string `json:"url"`
		} `json:"logo"`
	} `json:"data"`
}

type modrinthSearchAPIResponse struct {
	Hits []struct {
		Slug        string `json:"slug"`
		Title       string `json:"title"`
		Description string `json:"description"`
		ProjectID   string `json:"project_id"`
		IconURL     string `json:"icon_url"`
		Downloads   int64  `json:"downloads"`
		Follows     int64  `json:"follows"`
	} `json:"hits"`
	TotalHits int `json:"total_hits"`
}

type modrinthVersion struct {
	ID           string   `json:"id"`
	GameVersions []string `json:"game_versions"`
	Loaders      []string `json:"loaders"`
	Files        []struct {
		URL      string `json:"url"`
		Filename string `json:"filename"`
		Primary  bool   `json:"primary"`
	} `json:"files"`
}

type curseForgeFilesResponse struct {
	Data []struct {
		ID       int64  `json:"id"`
		FileName string `json:"fileName"`
	} `json:"data"`
}

func httpUserAgent() string {
	return "QMLauncher/" + version.Current
}

// remoteStoreHTTPClient is used for CurseForge / Modrinth catalog HTTP (debug-traced when launcher_debug is on).
var remoteStoreHTTPClient = network.HTTPClientForExternal(60 * time.Second)

func httpGetJSON(u string, headers map[string]string, out any) error {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", httpUserAgent())
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := remoteStoreHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		body := strings.TrimSpace(string(slurp))
		if resp.StatusCode == http.StatusForbidden && strings.Contains(u, "api.curseforge.com") {
			notifyCurseForgeAPI403()
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// curseForgeKey403Hint appends a short troubleshooting note when CurseForge rejects the API key.
func curseForgeKey403Hint(err error) error {
	if err == nil {
		return err
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, "HTTP 403") {
		return err
	}
	hint := "CurseForge отклонил ваш API токен"
	return fmt.Errorf("%s — %s", msg, hint)
}

// RemoteStoreCategory maps UI tabs to backend facets / classes.
func curseForgeClassID(category string) int {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "resourcepacks":
		return cfClassRP
	case "shaderpacks":
		return cfClassShaders
	case "datapacks":
		return cfClassDatapacks
	case "modpacks":
		return cfClassModpacks
	default:
		return cfClassMods
	}
}

// curseForgeSortField: popularity=2, downloads=6 (ModsSearchSortField).
func curseForgeSortField(sortName string) int {
	switch strings.ToLower(strings.TrimSpace(sortName)) {
	case "downloads":
		return 6
	default:
		return 2
	}
}

func modrinthProjectType(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "resourcepacks":
		return "resourcepack"
	case "shaderpacks":
		return "shader"
	case "datapacks":
		return "datapack"
	case "modpacks":
		return "modpack"
	default:
		return "mod"
	}
}

func modrinthIndex(sortName string) string {
	switch strings.ToLower(strings.TrimSpace(sortName)) {
	case "downloads":
		return "downloads"
	default:
		return "follows"
	}
}

func cfwidgetCurseForgeSegmentsForCategory(category string) []string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "modpacks":
		return []string{"modpacks", "mc-mods"}
	case "resourcepacks":
		return []string{"texture-packs"}
	case "shaderpacks":
		return []string{"shaders"}
	case "datapacks":
		return []string{"mc-mods"}
	default:
		return []string{"mc-mods"}
	}
}

func slugFromCurseForgePageURL(cfURL string) string {
	cfURL = strings.TrimSpace(cfURL)
	if cfURL == "" {
		return ""
	}
	u, err := url.Parse(cfURL)
	if err != nil {
		return ""
	}
	segs := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segs) == 0 {
		return ""
	}
	return segs[len(segs)-1]
}

// cfwidgetHitForModrinthSlug loads CurseForge project metadata via CFWidget when the Modrinth slug matches CurseForge (public mirror; no official API key).
func cfwidgetHitForModrinthSlug(category, mrSlug string, mrDownloads int64) *RemoteStoreHit {
	mrSlug = strings.TrimSpace(mrSlug)
	if mrSlug == "" {
		return nil
	}
	for _, seg := range cfwidgetCurseForgeSegmentsForCategory(category) {
		u := fmt.Sprintf("%s/minecraft/%s/%s", cfwidgetAPIBase, seg, url.PathEscape(mrSlug))
		var raw cfwidgetProjectResp
		if err := httpGetJSON(u, nil, &raw); err != nil {
			continue
		}
		if raw.ID <= 0 {
			continue
		}
		title := strings.TrimSpace(raw.Title)
		if title == "" {
			continue
		}
		pageURL := strings.TrimSpace(raw.URLs.Curseforge)
		slug := slugFromCurseForgePageURL(pageURL)
		if slug == "" {
			slug = mrSlug
		}
		icon := strings.TrimSpace(raw.Thumbnail)
		downloads := raw.Downloads.Total
		if downloads == 0 && mrDownloads > 0 {
			downloads = mrDownloads
		}
		return &RemoteStoreHit{
			Source:    "curseforge",
			ProjectID: strconv.Itoa(raw.ID),
			Slug:      slug,
			Title:     title,
			Summary:   trimSummary(strings.TrimSpace(raw.Summary), 220),
			IconURL:   icon,
			PageURL:   pageURL,
			Downloads: downloads,
		}
	}
	return nil
}

func searchCurseForgeStoreCfWidgetBridge(category, query, sort string, page, pageSize int, cachesDir string) ([]RemoteStoreHit, error) {
	mrHits, err := SearchModrinthStore(category, query, sort, page, pageSize, cachesDir)
	if err != nil {
		return nil, err
	}
	var out []RemoteStoreHit
	for _, mr := range mrHits {
		if h := cfwidgetHitForModrinthSlug(category, mr.Slug, mr.Downloads); h != nil {
			out = append(out, *h)
		}
	}
	return out, nil
}

// SearchCurseForgeStore searches CurseForge Core API (requires x-api-key; same key as downloads).
func SearchCurseForgeStore(category, query, sort string, page, pageSize int, cachesDir string) ([]RemoteStoreHit, error) {
	if debuglog.Enabled() {
		k := CurseForgeAPIKey()
		debuglog.Printf("CurseForge: SearchCurseForgeStore category=%q query=%q page=%d pageSize=%d effectiveApiKeyLen=%d", category, strings.TrimSpace(query), page, pageSize, len(k))
	}
	if strings.TrimSpace(CurseForgeAPIKey()) == "" {
		if debuglog.Enabled() {
			debuglog.Printf("CurseForge: SearchCurseForgeStore using Modrinth + CFWidget bridge (no API key)")
		}
		return searchCurseForgeStoreCfWidgetBridge(category, query, sort, page, pageSize, cachesDir)
	}
	_ = cachesDir // reserved for disk cache; fetch always uses authenticated API
	classID := curseForgeClassID(category)
	sortF := curseForgeSortField(sort)
	idx := 0
	if page > 0 {
		idx = page * pageSize
	}
	u := fmt.Sprintf(
		"https://api.curseforge.com/v1/mods/search?gameId=%d&classId=%d&searchFilter=%s&sortField=%d&sortOrder=desc&pageSize=%d&index=%d",
		minecraftGameID,
		classID,
		url.QueryEscape(strings.TrimSpace(query)),
		sortF,
		pageSize,
		idx,
	)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		apiKey := CurseForgeAPIKey()
		if apiKey == "" {
			return nil, fmt.Errorf("не удалось выполнить поиск на CurseForge")
		}
		var raw curseForgeSearchAPIResponse
		err := httpGetJSON(u, map[string]string{"x-api-key": apiKey}, &raw)
		if err == nil {
			var out []RemoteStoreHit
			for _, d := range raw.Data {
				icon := d.Logo.ThumbnailURL
				if icon == "" {
					icon = d.Logo.URL
				}
				pageLink := d.Links.WebsiteURL
				if pageLink == "" && d.Slug != "" {
					pageLink = curseForgeWebPage(category, d.ID, d.Slug)
				}
				out = append(out, RemoteStoreHit{
					Source:    "curseforge",
					ProjectID: strconv.Itoa(d.ID),
					Slug:      d.Slug,
					Title:     d.Name,
					Summary:   trimSummary(d.Summary, 220),
					IconURL:   icon,
					PageURL:   pageLink,
					Downloads: d.DownloadCount,
				})
			}
			return out, nil
		}
		lastErr = err
		if attempt == 0 && strings.HasPrefix(err.Error(), "HTTP 403:") {
			continue
		}
		break
	}
	return nil, curseForgeKey403Hint(lastErr)
}

func curseForgeWebPage(category string, id int, slug string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "resourcepacks":
		if slug != "" {
			return "https://www.curseforge.com/minecraft/texture-packs/" + slug
		}
	case "modpacks":
		if slug != "" {
			return "https://www.curseforge.com/minecraft/modpacks/" + slug
		}
	case "shaderpacks", "datapacks":
		return fmt.Sprintf("https://www.curseforge.com/minecraft/mc-mods/%d", id)
	}
	if slug != "" {
		return "https://www.curseforge.com/minecraft/mc-mods/" + slug
	}
	return fmt.Sprintf("https://www.curseforge.com/minecraft/mc-mods/%d", id)
}

func trimSummary(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func safeCacheKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "empty"
	}
	repl := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_")
	return repl.Replace(s)
}

// SearchModrinthStore searches Modrinth with project_type facet.
func SearchModrinthStore(category, query, sort string, page, pageSize int, cachesDir string) ([]RemoteStoreHit, error) {
	pt := modrinthProjectType(category)
	idx := modrinthIndex(sort)
	offset := 0
	if page > 0 {
		offset = page * pageSize
	}
	uu, err := url.Parse("https://api.modrinth.com/v2/search")
	if err != nil {
		return nil, err
	}
	q := uu.Query()
	q.Set("query", strings.TrimSpace(query))
	q.Set("limit", strconv.Itoa(pageSize))
	q.Set("offset", strconv.Itoa(offset))
	q.Set("index", idx)
	q.Set("facets", fmt.Sprintf(`[["project_type:%s"]]`, pt))
	uu.RawQuery = q.Encode()
	if debuglog.Enabled() {
		debuglog.Printf("Modrinth: SearchModrinthStore category=%q query=%q page=%d pageSize=%d url=%s", category, strings.TrimSpace(query), page, pageSize, uu.String())
	}

	cachePath := filepath.Join(cachesDir, "modrinth", fmt.Sprintf("store_%s_%s_%s_o%d.json", pt, idx, safeCacheKey(query), offset))
	cache := network.Cache[modrinthSearchAPIResponse]{
		Path:        cachePath,
		URL:         uu.String(),
		AlwaysFetch: true,
	}
	var raw modrinthSearchAPIResponse
	if err := cache.Get(&raw); err != nil {
		return nil, err
	}
	var out []RemoteStoreHit
	for _, h := range raw.Hits {
		out = append(out, RemoteStoreHit{
			Source:    "modrinth",
			ProjectID: h.ProjectID,
			Slug:      h.Slug,
			Title:     h.Title,
			Summary:   trimSummary(h.Description, 220),
			IconURL:   h.IconURL,
			PageURL:   "https://modrinth.com/" + modrinthPathSegment(category) + "/" + h.Slug,
			Downloads: h.Downloads,
		})
	}
	return out, nil
}

func modrinthPathSegment(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "resourcepacks":
		return "resourcepack"
	case "shaderpacks":
		return "shader"
	case "datapacks":
		return "datapack"
	case "modpacks":
		return "modpack"
	default:
		return "mod"
	}
}

func normalizeStoreTitle(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" {
		return ""
	}
	return strings.Join(strings.Fields(title), " ")
}

func remoteStoreSideFrom(h RemoteStoreHit) *RemoteStoreSide {
	return &RemoteStoreSide{
		ProjectID: h.ProjectID,
		Slug:      h.Slug,
		PageURL:   h.PageURL,
		Downloads: h.Downloads,
	}
}

func pickLongerSummary(a, b string) string {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if len(b) > len(a) {
		return b
	}
	return a
}

func pickNonEmptyIconURL(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func pickDisplayTitle(cf, mr RemoteStoreHit) string {
	if len(strings.TrimSpace(cf.Title)) >= len(strings.TrimSpace(mr.Title)) {
		return cf.Title
	}
	return mr.Title
}

// mergeBucket is one logical row while merging CF + Modrinth lists.
type mergeBucket struct {
	cf *RemoteStoreHit
	mr *RemoteStoreHit
}

// MergeRemoteStoreHits combines CurseForge and Modrinth results: same normalized
// title becomes one card (source "both") with cf+mr details; order follows the
// previous interleave (one CF, then one MR per step) for first appearance of each row key.
func MergeRemoteStoreHits(cfHits, mrHits []RemoteStoreHit) []RemoteStoreHit {
	buckets := make(map[string]*mergeBucket)
	orderSeen := make(map[string]bool)
	var order []string

	appendOrder := func(k string) {
		if orderSeen[k] {
			return
		}
		orderSeen[k] = true
		order = append(order, k)
	}

	placeCf := func(h RemoteStoreHit) {
		k := normalizeStoreTitle(h.Title)
		if k == "" {
			k = "_cf_" + h.ProjectID
		}
		b := buckets[k]
		if b != nil && b.cf != nil {
			k = k + "|cf|" + h.ProjectID
			b = buckets[k]
		}
		if b == nil {
			b = &mergeBucket{}
			buckets[k] = b
		}
		b.cf = &h
		appendOrder(k)
	}

	placeMr := func(h RemoteStoreHit) {
		k := normalizeStoreTitle(h.Title)
		if k == "" {
			k = "_mr_" + h.ProjectID
		}
		b := buckets[k]
		if b != nil && b.mr != nil {
			k = k + "|mr|" + h.ProjectID
			b = buckets[k]
		}
		if b == nil {
			b = &mergeBucket{}
			buckets[k] = b
		}
		b.mr = &h
		appendOrder(k)
	}

	i, j := 0, 0
	for i < len(cfHits) || j < len(mrHits) {
		if i < len(cfHits) {
			placeCf(cfHits[i])
			i++
		}
		if j < len(mrHits) {
			placeMr(mrHits[j])
			j++
		}
	}

	out := make([]RemoteStoreHit, 0, len(order))
	for _, k := range order {
		b := buckets[k]
		if b == nil {
			continue
		}
		switch {
		case b.cf != nil && b.mr != nil:
			cf, mr := *b.cf, *b.mr
			out = append(out, RemoteStoreHit{
				Source:    "both",
				Title:     pickDisplayTitle(cf, mr),
				Summary:   pickLongerSummary(cf.Summary, mr.Summary),
				IconURL:   pickNonEmptyIconURL(cf.IconURL, mr.IconURL),
				PageURL:   pickNonEmptyIconURL(cf.PageURL, mr.PageURL),
				Downloads: cf.Downloads + mr.Downloads,
				Cf:        remoteStoreSideFrom(cf),
				Mr:        remoteStoreSideFrom(mr),
			})
		case b.cf != nil:
			h := *b.cf
			out = append(out, h)
		default:
			h := *b.mr
			out = append(out, h)
		}
	}
	return out
}

func normalizeModrinthLoaders(ldr string) []string {
	l := strings.ToLower(strings.TrimSpace(ldr))
	switch l {
	case "fabric":
		return []string{"fabric"}
	case "quilt":
		// Многие проекты помечены только как fabric; Quilt их подхватывает.
		return []string{"quilt", "fabric"}
	case "forge":
		return []string{"forge"}
	case "neoforge":
		return []string{"neoforge"}
	default:
		return nil
	}
}

// remoteStoreCategoryUsesModLoader — для модов и модпаков учитываем загрузчик инстанса.
func remoteStoreCategoryUsesModLoader(category string) bool {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "mods", "modpacks":
		return true
	default:
		return false
	}
}

// curseForgeModLoaderType: CurseForge API ModLoaderType (0=any … 6=NeoForge).
func curseForgeModLoaderType(loader string) int {
	switch strings.ToLower(strings.TrimSpace(loader)) {
	case "forge":
		return 1
	case "fabric":
		return 4
	case "quilt":
		return 5
	case "neoforge":
		return 6
	default:
		return 0
	}
}

func fetchCurseForgeModFiles(modID int64, gameVersion string, modLoaderType int, applyLoader bool, apiKey string) (curseForgeFilesResponse, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.curseforge.com",
		Path:   fmt.Sprintf("/v1/mods/%d/files", modID),
	}
	q := u.Query()
	q.Set("pageSize", "50")
	q.Set("sortOrder", "desc")
	if gameVersion != "" {
		q.Set("gameVersion", gameVersion)
	}
	if applyLoader && modLoaderType > 0 && gameVersion != "" {
		q.Set("modLoaderType", strconv.Itoa(modLoaderType))
	}
	u.RawQuery = q.Encode()
	var raw curseForgeFilesResponse
	if err := httpGetJSON(u.String(), map[string]string{"x-api-key": apiKey}, &raw); err != nil {
		return raw, err
	}
	return raw, nil
}

func mrVersionListsGame(v modrinthVersion, game string) bool {
	game = strings.TrimSpace(game)
	if game == "" {
		return true
	}
	for _, g := range v.GameVersions {
		if strings.EqualFold(strings.TrimSpace(g), game) {
			return true
		}
	}
	return false
}

func mrVersionListsLoader(v modrinthVersion, want []string) bool {
	if len(want) == 0 {
		return true
	}
	for _, w := range want {
		for _, lv := range v.Loaders {
			if strings.EqualFold(lv, w) {
				return true
			}
		}
	}
	return false
}

func pickModrinthFile(v modrinthVersion) (fileURL, filename string) {
	for _, f := range v.Files {
		if f.Primary && f.URL != "" {
			return f.URL, f.Filename
		}
	}
	for _, f := range v.Files {
		if f.URL != "" {
			return f.URL, f.Filename
		}
	}
	return "", ""
}

// DownloadModrinthProjectTo writes the best-matching primary file into destDir using the remote file name.
// category: mods и modpacks — фильтр по gameVersion и загрузчику инстанса; остальное — в основном по версии игры.
func DownloadModrinthProjectTo(projectSlug, gameVersion, loader, category, destDir string) (savedPath string, err error) {
	projectSlug = strings.TrimSpace(projectSlug)
	if projectSlug == "" {
		return "", fmt.Errorf("empty Modrinth project")
	}
	u := "https://api.modrinth.com/v2/project/" + url.PathEscape(projectSlug) + "/version"
	var versions []modrinthVersion
	if err := httpGetJSON(u, nil, &versions); err != nil {
		return "", err
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions for project %s", projectSlug)
	}
	gameVersion = strings.TrimSpace(gameVersion)
	loader = strings.TrimSpace(loader)
	filterLoader := remoteStoreCategoryUsesModLoader(category)
	loaders := normalizeModrinthLoaders(loader)

	var chosen *modrinthVersion
	switch {
	case filterLoader && len(loaders) > 0 && gameVersion != "":
		for i := range versions {
			v := &versions[i]
			if mrVersionListsGame(*v, gameVersion) && mrVersionListsLoader(*v, loaders) {
				chosen = v
				break
			}
		}
		if chosen == nil {
			return "", fmt.Errorf("на Modrinth нет сборки для Minecraft %s и загрузчика %s (проект %s)", gameVersion, loader, projectSlug)
		}
	case filterLoader && len(loaders) > 0 && gameVersion == "":
		return "", fmt.Errorf("в инстансе не указана версия Minecraft — нужна для выбора файла мода на Modrinth")
	case filterLoader && len(loaders) == 0:
		if gameVersion == "" {
			return "", fmt.Errorf("в инстансе не указаны версия Minecraft или поддерживаемый загрузчик для мода с Modrinth")
		}
		for i := range versions {
			v := &versions[i]
			if mrVersionListsGame(*v, gameVersion) {
				chosen = v
				break
			}
		}
		if chosen == nil {
			return "", fmt.Errorf("на Modrinth нет файла для Minecraft %s (проект %s)", gameVersion, projectSlug)
		}
	case gameVersion != "":
		for i := range versions {
			v := &versions[i]
			if mrVersionListsGame(*v, gameVersion) {
				chosen = v
				break
			}
		}
		if chosen == nil {
			return "", fmt.Errorf("на Modrinth нет файла для Minecraft %s (проект %s)", gameVersion, projectSlug)
		}
	default:
		chosen = &versions[0]
	}
	fileURL, fname := pickModrinthFile(*chosen)
	if fileURL == "" {
		return "", fmt.Errorf("no downloadable file for %s", projectSlug)
	}
	if fname == "" {
		fname = projectSlug + "-download"
	}
	destPath := filepath.Join(destDir, filepath.Base(fname))
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}
	if err := network.DownloadFile(network.DownloadEntry{URL: fileURL, Path: destPath}); err != nil {
		return "", err
	}
	return destPath, nil
}

// DownloadCurseForgeProjectTo выбирает файл по версии игры и загрузчику инстанса (API CurseForge).
// category: mods и modpacks — передаётся modLoaderType вместе с gameVersion.
func DownloadCurseForgeProjectTo(modIDStr, gameVersion, loader, category, apiKey, destDir string) (savedPath string, err error) {
	modID, err := strconv.ParseInt(strings.TrimSpace(modIDStr), 10, 64)
	if err != nil {
		return "", fmt.Errorf("curseforge mod id: %w", err)
	}
	gameVersion = strings.TrimSpace(gameVersion)
	useLoader := remoteStoreCategoryUsesModLoader(category)
	if useLoader && gameVersion == "" {
		return "", fmt.Errorf("в инстансе не указана версия Minecraft — нужна для выбора файла на CurseForge")
	}
	mlt := curseForgeModLoaderType(loader)
	if debuglog.Enabled() {
		debuglog.Printf("CurseForge: DownloadCurseForgeProjectTo modID=%d gameVersion=%q loader=%q category=%q modLoaderType=%d", modID, gameVersion, loader, category, mlt)
	}

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		k := NormalizeCurseForgeAPIKey(strings.TrimSpace(apiKey))
		if attempt > 0 {
			k = CurseForgeAPIKey()
		}
		if k == "" {
			return "", fmt.Errorf("не удалось загрузить файл с CurseForge")
		}

		listing, err1 := fetchCurseForgeModFiles(modID, gameVersion, mlt, useLoader, k)
		if err1 != nil {
			lastErr = err1
			if attempt == 0 && strings.HasPrefix(err1.Error(), "HTTP 403:") {
				continue
			}
			return "", curseForgeKey403Hint(err1)
		}
		if len(listing.Data) == 0 {
			if useLoader && gameVersion != "" && mlt > 0 {
				return "", fmt.Errorf("на CurseForge нет файла для Minecraft %s и загрузчика %s (проект %d)", gameVersion, strings.TrimSpace(loader), modID)
			}
			if gameVersion != "" {
				return "", fmt.Errorf("на CurseForge нет файла для Minecraft %s (проект %d)", gameVersion, modID)
			}
			return "", fmt.Errorf("нет файлов на CurseForge для проекта %d", modID)
		}
		fileID := listing.Data[0].ID
		baseName := listing.Data[0].FileName
		if baseName == "" {
			baseName = fmt.Sprintf("%d-file", fileID)
		}
		dlURL := fmt.Sprintf("https://api.curseforge.com/v1/mods/%d/files/%d/download-url", modID, fileID)
		var dlPayload struct {
			Data string `json:"data"`
		}
		if err2 := httpGetJSON(dlURL, map[string]string{"x-api-key": k}, &dlPayload); err2 != nil {
			lastErr = err2
			if attempt == 0 && strings.HasPrefix(err2.Error(), "HTTP 403:") {
				continue
			}
			return "", curseForgeKey403Hint(err2)
		}
		if dlPayload.Data == "" {
			return "", fmt.Errorf("пустой download-url от CurseForge")
		}
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return "", err
		}
		destPath := filepath.Join(destDir, filepath.Base(baseName))
		if err := network.DownloadFile(network.DownloadEntry{URL: dlPayload.Data, Path: destPath}); err != nil {
			return "", err
		}
		return destPath, nil
	}
	if lastErr != nil {
		return "", curseForgeKey403Hint(lastErr)
	}
	return "", fmt.Errorf("не удалось загрузить файл с CurseForge")
}
