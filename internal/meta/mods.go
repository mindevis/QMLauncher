package meta

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"QMLauncher/internal/network"
)

// ModInfo represents information about a mod
type ModInfo struct {
	Name          string
	Slug          string
	CurseForgeID  string
	ModrinthID    string
	CurseForgeURL string
	ModrinthURL   string
}

// ResourcePackInfo represents information about a resource pack
type ResourcePackInfo struct {
	Name          string
	Slug          string
	CurseForgeID  string
	ModrinthID    string
	CurseForgeURL string
	ModrinthURL   string
}

// CurseForgeSearchResult represents the response from CurseForge API
type CurseForgeSearchResult struct {
	Data []struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Slug  string `json:"slug"`
		Links struct {
			WebsiteURL string `json:"websiteUrl"`
		} `json:"links"`
	} `json:"data"`
}

// ModrinthSearchResult represents the response from Modrinth API
type ModrinthSearchResult struct {
	Hits []struct {
		Slug        string   `json:"slug"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Categories  []string `json:"categories"`
		Gallery     []string `json:"gallery"`
		ProjectID   string   `json:"project_id"`
	} `json:"hits"`
}

// SearchModOnCurseForge searches for a mod on CurseForge using alternative methods
func SearchModOnCurseForge(modName string, cachesDir string) (string, error) {
	// Try alternative CurseForge search methods

	// Method 1: Try the public API endpoint without authentication
	// Some endpoints might work without API key
	searchURL := fmt.Sprintf("https://api.curseforge.com/v1/mods/search?gameId=432&searchFilter=%s&pageSize=1&sortField=2&sortOrder=desc", url.QueryEscape(modName))

	cache := network.Cache[CurseForgeSearchResult]{
		Path:        filepath.Join(cachesDir, "curseforge", "search_"+strings.ReplaceAll(modName, " ", "_")+".json"),
		URL:         searchURL,
		AlwaysFetch: false,
	}

	var result CurseForgeSearchResult
	if err := cache.Get(&result); err == nil {
		if len(result.Data) > 0 {
			modID := fmt.Sprintf("%d", result.Data[0].ID)
			url := fmt.Sprintf("https://www.curseforge.com/minecraft/mc-mods/%s", modID)
			return url, nil
		}
	}

	// Method 2: If API fails, try to construct URL based on common patterns
	// Many mods follow predictable URL patterns
	// This is a fallback method
	return constructCurseForgeURL(modName)
}

// constructCurseForgeURL creates a CurseForge URL based on common naming patterns
func constructCurseForgeURL(modName string) (string, error) {
	// Known correct CurseForge URLs for specific mods
	knownURLs := map[string]string{
		"xaeros-world-map":        "https://www.curseforge.com/minecraft/mc-mods/xaeros-world-map",
		"xaeros-minimap":          "https://www.curseforge.com/minecraft/mc-mods/xaeros-minimap",
		"catalogue":               "https://www.curseforge.com/minecraft/mc-mods/catalogue",
		"configured":              "https://www.curseforge.com/minecraft/mc-mods/configured",
		"just-enough-items-jei":   "https://www.curseforge.com/minecraft/mc-mods/just-enough-items-jei",
		"open-parties-and-claims": "https://www.curseforge.com/minecraft/mc-mods/open-parties-and-claims",
	}

	// Check if we have a known URL
	if url, exists := knownURLs[modName]; exists {
		return url, nil
	}

	// Fallback: generate URL from mod name
	cleanName := strings.ToLower(strings.ReplaceAll(modName, " ", "-"))
	url := fmt.Sprintf("https://www.curseforge.com/minecraft/mc-mods/%s", cleanName)
	return url, nil
}

// SearchModOnModrinth searches for a mod on Modrinth
func SearchModOnModrinth(modName string, cachesDir string) (string, error) {
	return SearchModOnModrinthWithCache(modName, cachesDir, "", "")
}

// SearchModOnModrinthWithLoader searches for a mod on Modrinth with loader and version filtering
func SearchModOnModrinthWithLoader(modName string, cachesDir string, loader string, gameVersion string) (string, error) {
	return SearchModOnModrinthWithCache(modName, cachesDir, loader, gameVersion)
}

// ModCacheEntry represents cached information about a mod
type ModCacheEntry struct {
	CurseForgeURL string `json:"curseforge_url,omitempty"`
	ModrinthID    string `json:"modrinth_id,omitempty"`
	LastChecked   int64  `json:"last_checked"`
}

// ModCache represents the cache structure for mod mappings
type ModCache struct {
	Mods map[string]ModCacheEntry `json:"mods"`
}

// SearchModOnModrinthWithCache searches for a mod on Modrinth with persistent caching
func SearchModOnModrinthWithCache(modName string, cachesDir string, loader string, gameVersion string) (string, error) {
	// First, check our persistent mod cache
	cachePath := filepath.Join(cachesDir, "modrinth_mods_cache.json")
	var modCache ModCache

	// Try to load existing cache
	if cacheData, err := os.ReadFile(cachePath); err == nil {
		if err := json.Unmarshal(cacheData, &modCache); err != nil {
			// If cache is corrupted, start fresh
			modCache = ModCache{Mods: make(map[string]ModCacheEntry)}
		}
	} else {
		modCache = ModCache{Mods: make(map[string]ModCacheEntry)}
	}

	// Create cache key that includes loader and version for better specificity
	cacheKey := modName
	if loader != "" && gameVersion != "" {
		cacheKey = fmt.Sprintf("%s_%s_%s", modName, loader, gameVersion)
	}

	// Check if we have cached result (only if it's recent and has data)
	if entry, exists := modCache.Mods[cacheKey]; exists {
		// If we have a recent cached result with data, return it
		if entry.ModrinthID != "" && time.Now().Unix()-entry.LastChecked < 86400 { // 24 hours
			return entry.ModrinthID, nil
		}
		// If cached result is empty or old, we'll search again
	}

	// Search via API (without strict filtering, we'll check compatibility after)
	searchURL := fmt.Sprintf("https://api.modrinth.com/v2/search?query=%s&limit=20", url.QueryEscape(modName))

	// Note: We'll check loader and version compatibility after getting results

	apiCache := network.Cache[ModrinthSearchResult]{
		Path:        filepath.Join(cachesDir, "modrinth", "search_"+strings.ReplaceAll(cacheKey, " ", "_")+".json"),
		URL:         searchURL,
		AlwaysFetch: false,
	}

	var result ModrinthSearchResult
	if err := apiCache.Get(&result); err != nil {
		// Cache empty result to avoid repeated failed searches
		modCache.Mods[cacheKey] = ModCacheEntry{
			LastChecked: time.Now().Unix(),
		}
		saveModCache(cachePath, modCache)
		return "", fmt.Errorf("failed to search Modrinth: %w", err)
	}
	// Look for matches with loader/version compatibility check
	var foundID string
	for _, hit := range result.Hits {
		// Check for exact slug match
		if strings.EqualFold(hit.Slug, modName) {
			// If we found an exact match, check compatibility
			if isCompatible(hit.ProjectID, loader, gameVersion, cachesDir) {
				foundID = hit.ProjectID
				break
			}
		}

		// Check for exact title match
		if strings.EqualFold(hit.Title, modName) {
			// If we found an exact match, check compatibility
			if isCompatible(hit.ProjectID, loader, gameVersion, cachesDir) {
				foundID = hit.ProjectID
				break
			}
		}

		// Check for special cases - JEI abbreviation
		if foundID == "" && strings.EqualFold(modName, "just-enough-items-jei") {
			titleLower := strings.ToLower(hit.Title)
			if strings.Contains(titleLower, "just enough items") && strings.Contains(titleLower, "jei") {
				if isCompatible(hit.ProjectID, loader, gameVersion, cachesDir) {
					foundID = hit.ProjectID
					break
				}
			}
		}
	}

	// If no exact match found, leave empty

	// Cache the result
	modCache.Mods[cacheKey] = ModCacheEntry{
		ModrinthID:  foundID,
		LastChecked: time.Now().Unix(),
	}
	saveModCache(cachePath, modCache)

	return foundID, nil
}

// saveModCache saves the mod cache to disk
func saveModCache(cachePath string, modCache ModCache) {
	if cacheData, err := json.MarshalIndent(modCache, "", "  "); err == nil {
		os.MkdirAll(filepath.Dir(cachePath), 0755)
		os.WriteFile(cachePath, cacheData, 0644)
	}
}

// ModrinthProject represents the structure of a Modrinth project API response
type ModrinthProject struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Loaders     []string `json:"loaders"`
	Versions    []string `json:"versions"`
}

// isCompatible checks if a mod is compatible with the given loader and game version
func isCompatible(projectID, loader, gameVersion, cachesDir string) bool {
	// For now, just check loader compatibility, version checking is more complex
	if loader == "" {
		return true // No filtering needed
	}

	// Get project details from Modrinth API
	projectURL := fmt.Sprintf("https://api.modrinth.com/v2/project/%s", projectID)
	cache := network.Cache[ModrinthProject]{
		Path:        filepath.Join(cachesDir, "modrinth", "project_"+projectID+".json"),
		URL:         projectURL,
		AlwaysFetch: false,
	}

	var project ModrinthProject
	if err := cache.Get(&project); err != nil {
		return false // If we can't get project info, assume incompatible
	}

	// Check loader compatibility
	for _, projLoader := range project.Loaders {
		if strings.EqualFold(projLoader, loader) {
			return true
		}
	}
	return false
}

// ExtractModInfoFromFilename extracts mod information from JAR filename
func ExtractModInfoFromFilename(filename string) ModInfo {
	// Remove .jar extension
	name := strings.TrimSuffix(filename, ".jar")

	var modInfo ModInfo
	modInfo.Name = filename

	// Clean the name by removing loader-specific suffixes/prefixes first
	cleanName := name
	cleanName = strings.TrimSuffix(cleanName, "-neoforge")
	cleanName = strings.TrimSuffix(cleanName, "-forge")
	cleanName = strings.TrimSuffix(cleanName, "-fabric")
	cleanName = strings.TrimSuffix(cleanName, "-quilt")
	cleanName = strings.TrimPrefix(cleanName, "neoforge-")
	cleanName = strings.TrimPrefix(cleanName, "forge-")
	cleanName = strings.TrimPrefix(cleanName, "fabric-")
	cleanName = strings.TrimPrefix(cleanName, "quilt-")

	// Known mod name mappings (filename -> correct slug)
	knownMods := map[string]string{
		"XaerosWorldMap":                  "xaeros-world-map",
		"XaerosWorldMap_1.39.12_NeoForge": "xaeros-world-map",
		"Xaeros_Minimap":                  "xaeros-minimap",
		"Xaeros_Minimap_25.2.10_NeoForge": "xaeros-minimap",
		"catalogue":                       "catalogue",
		"configured":                      "configured",
		"jei":                             "just-enough-items-jei",
		"open-parties-and-claims":         "open-parties-and-claims",
	}

	// Multi-step approach to clean mod names
	workingName := cleanName

	// Step 1: Find loader and take everything before it
	loaders := []string{"neoforge", "forge", "fabric", "quilt"}
	for _, loader := range loaders {
		if idx := strings.Index(workingName, loader); idx > 0 {
			workingName = strings.TrimRight(workingName[:idx], "-_ ")
			break
		}
	}

	// Step 2: Clean up remaining version patterns
	// Handle cases like "XaerosWorldMap_1.39.12" -> "XaerosWorldMap"
	if strings.Contains(workingName, "_") {
		parts := strings.Split(workingName, "_")
		// Go backwards and remove version-like parts
		for i := len(parts) - 1; i >= 0; i-- {
			part := parts[i]
			if len(part) > 0 && (part[0] >= '0' && part[0] <= '9') {
				// This looks like a version, remove it
				parts = parts[:i]
			} else {
				break
			}
		}
		workingName = strings.Join(parts, "_")
	}

	// Step 3: Handle dash-separated versions
	if strings.Contains(workingName, "-") {
		parts := strings.Split(workingName, "-")
		// Go backwards and remove version-like parts
		for i := len(parts) - 1; i >= 0; i-- {
			part := parts[i]
			if len(part) > 0 && (part[0] >= '0' && part[0] <= '9') {
				// This looks like a version, remove it
				parts = parts[:i]
			} else {
				break
			}
		}
		workingName = strings.Join(parts, "-")
	}

	// Step 4: Check if we have a known mod mapping
	if mappedName, exists := knownMods[workingName]; exists {
		modInfo.Slug = mappedName
	} else {
		modInfo.Slug = workingName
	}

	return modInfo
}

// GetModLinks searches for mod links on CurseForge and Modrinth
func GetModLinks(modInfo ModInfo, cachesDir string, loader string, gameVersion string) ModInfo {

	// Try CurseForge first
	if curseURL, err := SearchModOnCurseForge(modInfo.Slug, cachesDir); err == nil && curseURL != "" {
		modInfo.CurseForgeURL = curseURL
		// Extract ID from URL if it's a numeric ID
		if strings.Contains(curseURL, "/mc-mods/") {
			parts := strings.Split(curseURL, "/mc-mods/")
			if len(parts) > 1 && len(parts[1]) > 0 && parts[1][0] >= '0' && parts[1][0] <= '9' {
				modInfo.CurseForgeID = strings.Split(parts[1], "/")[0]
			}
		}
	}

	// Try Modrinth with loader and version filtering
	if modrinthID, err := SearchModOnModrinthWithCache(modInfo.Slug, cachesDir, loader, gameVersion); err == nil && modrinthID != "" {
		modInfo.ModrinthID = modrinthID
		modInfo.ModrinthURL = fmt.Sprintf("https://modrinth.com/mod/%s", modrinthID)
	}

	return modInfo
}

// ExtractResourcePackInfo extracts resource pack information from filename
func ExtractResourcePackInfo(filename string) ResourcePackInfo {
	// Remove common extensions
	name := strings.TrimSuffix(filename, ".zip")
	name = strings.TrimSuffix(name, ".jar")

	// Clean the name by removing version patterns and other suffixes
	cleanName := name

	// Remove version patterns like -1.2.3, _1.2.3, etc.
	versionPattern := regexp.MustCompile(`[-_.]?(?:alpha|beta|rc)?\d+(?:[._-]\d+)*(?:[._-][a-zA-Z0-9]+)?$`)
	for {
		newName := versionPattern.ReplaceAllString(cleanName, "")
		if newName == cleanName {
			break
		}
		cleanName = newName
	}

	// Clean up any remaining trailing hyphens or underscores and version prefixes
	cleanSlug := strings.TrimRight(cleanName, "-_")

	// Remove common version prefixes like _V, _v, -v, etc.
	versionPrefixPattern := regexp.MustCompile(`[-_][vV]$`)
	cleanSlug = versionPrefixPattern.ReplaceAllString(cleanSlug, "")

	// Create human-readable name by converting underscores and cleaning up
	humanName := strings.ReplaceAll(cleanSlug, "_", " ")
	humanName = strings.ReplaceAll(humanName, "-", " ")
	humanName = strings.ReplaceAll(humanName, "+", " ")
	humanName = strings.Title(humanName) //nolint:staticcheck // Capitalize first letter of each word

	var rpInfo ResourcePackInfo
	rpInfo.Name = humanName
	rpInfo.Slug = cleanSlug

	return rpInfo
}

// GetResourcePackLinks searches for resource pack links on CurseForge and Modrinth
func GetResourcePackLinks(rpInfo ResourcePackInfo, cachesDir string, gameVersion string) ResourcePackInfo {
	// Skip CurseForge for now as links are broken (403 errors)
	// Try CurseForge with improved URL generation
	// if curseURL := SearchResourcePackOnCurseForge(rpInfo.Slug, cachesDir, gameVersion); curseURL != "" {
	//     rpInfo.CurseForgeURL = curseURL
	// }

	// Try Modrinth
	if modrinthID, err := SearchResourcePackOnModrinth(rpInfo.Slug, cachesDir, gameVersion); err == nil && modrinthID != "" {
		rpInfo.ModrinthID = modrinthID
		rpInfo.ModrinthURL = fmt.Sprintf("https://modrinth.com/resourcepack/%s", modrinthID)
	}

	return rpInfo
}

// SearchResourcePackOnCurseForge searches for a resource pack on CurseForge
func SearchResourcePackOnCurseForge(rpName string, cachesDir string, gameVersion string) string {
	// Create cache key
	cacheKey := fmt.Sprintf("rp_cf_%s_%s", rpName, gameVersion)
	cachePath := filepath.Join(cachesDir, "modrinth_resourcepacks_cache.json")

	// Try to load existing cache
	var rpCache map[string]string
	if cacheData, err := os.ReadFile(cachePath); err == nil {
		if err := json.Unmarshal(cacheData, &rpCache); err == nil {
			if url, exists := rpCache[cacheKey]; exists && url != "" {
				return url
			}
		}
	} else {
		rpCache = make(map[string]string)
	}

	// Known resource pack mappings (filename pattern -> CurseForge slug)
	// Note: Many CurseForge links are broken due to site changes, using Modrinth as primary source
	knownRPMappings := map[string]string{
		// Temporarily disabled until proper slugs are found
		// "Brewing_Guide_On_Minecraft":   "brewing-guide-on-minecraft",
		// "MandalasGUI+Dakmode":          "mandalas-gui-dark-mode",
		// "MandalasGUI_AddOn+DarkModded": "mandalas-gui-dark-mode", // fallback to main pack
	}

	// Check for known mappings first
	for pattern, slug := range knownRPMappings {
		if strings.Contains(rpName, pattern) {
			url := fmt.Sprintf("https://www.curseforge.com/minecraft/texture-packs/%s", slug)
			rpCache[cacheKey] = url
			saveRPCache(cachePath, rpCache)
			return url
		}
	}

	// Generate URL based on cleaned name as fallback
	cleanName := CleanResourcePackName(rpName)
	url := fmt.Sprintf("https://www.curseforge.com/minecraft/texture-packs/%s", cleanName)

	// Cache the result
	rpCache[cacheKey] = url
	saveRPCache(cachePath, rpCache)

	return url
}

// CleanResourcePackName cleans resource pack name for URL generation
func CleanResourcePackName(name string) string {
	// Remove file extensions
	name = strings.TrimSuffix(name, ".zip")
	name = strings.TrimSuffix(name, ".jar")

	// Replace problematic characters
	name = strings.ReplaceAll(name, "+", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")

	// Remove version patterns
	versionPattern := regexp.MustCompile(`-v?\d+(\.\d+)*.*$`)
	name = versionPattern.ReplaceAllString(name, "")

	// Clean up multiple dashes
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	// Remove trailing dashes
	name = strings.TrimSuffix(name, "-")

	return strings.ToLower(name)
}

// simplifyResourcePackNameForSearch creates a simplified search query from resource pack name
func simplifyResourcePackNameForSearch(name string) string {
	// Remove file extensions
	name = strings.TrimSuffix(name, ".zip")
	name = strings.TrimSuffix(name, ".jar")

	// Replace underscores and dashes with spaces
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "+", " ")

	// Remove version numbers and patterns
	versionPattern := regexp.MustCompile(`\bv?\d+(\.\d+)*\b`)
	name = versionPattern.ReplaceAllString(name, "")

	// Clean up extra spaces
	spacePattern := regexp.MustCompile(`\s+`)
	name = spacePattern.ReplaceAllString(name, " ")
	name = strings.TrimSpace(name)

	// Split into words and filter
	words := strings.Fields(strings.ToLower(name))
	filteredWords := []string{}

	// Common words to skip
	commonWords := map[string]bool{
		"gui": true, "addon": true, "add": true, "on": true, "dark": true,
		"mode": true, "modded": true, "pack": true, "resource": true,
		"texture": true, "v": true, "and": true, "the": true, "a": true,
	}

	for _, word := range words {
		if !commonWords[word] && len(word) > 2 {
			filteredWords = append(filteredWords, word)
		}
	}

	// Take first 2-3 significant words
	if len(filteredWords) > 3 {
		filteredWords = filteredWords[:3]
	} else if len(filteredWords) == 0 && len(words) > 0 {
		// Fallback: take first word even if it's common
		filteredWords = []string{words[0]}
	}

	return strings.Join(filteredWords, " ")
}

// SearchResourcePackOnModrinth searches for a resource pack on Modrinth
func SearchResourcePackOnModrinth(rpName string, cachesDir string, gameVersion string) (string, error) {

	// Create cache key for resource packs
	cacheKey := fmt.Sprintf("rp_%s_%s", rpName, gameVersion)

	// Check persistent cache
	cachePath := filepath.Join(cachesDir, "modrinth_resourcepacks_cache.json")
	var rpCache map[string]string

	// Try to load existing cache
	if cacheData, err := os.ReadFile(cachePath); err == nil {
		if err := json.Unmarshal(cacheData, &rpCache); err == nil {
			if id, exists := rpCache[cacheKey]; exists && id != "" {
				return id, nil
			}
		}
	} else {
		rpCache = make(map[string]string)
	}

	// Create a simplified search query from the resource pack name
	// Take first 2-3 words and clean them for better search results
	searchQuery := simplifyResourcePackNameForSearch(rpName)
	searchURL := fmt.Sprintf("https://api.modrinth.com/v2/search?query=%s&limit=10", url.QueryEscape(searchQuery))

	apiCache := network.Cache[ModrinthSearchResult]{
		Path:        filepath.Join(cachesDir, "modrinth", "search_rp_"+strings.ReplaceAll(cacheKey, " ", "_")+".json"),
		URL:         searchURL,
		AlwaysFetch: false,
	}

	var result ModrinthSearchResult
	if err := apiCache.Get(&result); err != nil {
		// Cache empty result
		rpCache[cacheKey] = ""
		saveRPCache(cachePath, rpCache)
		return "", fmt.Errorf("failed to search Modrinth for resource packs: %w", err)
	}

	// Known resource pack ID mappings (filename pattern -> Modrinth ID)
	knownRPMappings := map[string]string{
		"MandalasGUI+Dakmode":            "h6zxsNVF", // Mandala's GUI - Dark mode
		"MandalasGUI_AddOn+DarkModded":   "h6zxsNVF", // Same pack, fallback to main
		"MandalasGUI_AddOn+DarkModded_V": "h6zxsNVF", // Mandala's GUI - Dark mode (versioned addon)
	}

	// Check for known mappings first
	for pattern, id := range knownRPMappings {
		if strings.Contains(rpName, pattern) {
			rpCache[cacheKey] = id
			saveRPCache(cachePath, rpCache)
			return id, nil
		}
	}

	// Look for matches - be more flexible
	for _, hit := range result.Hits {
		titleLower := strings.ToLower(hit.Title)
		slugLower := strings.ToLower(hit.Slug)

		// Skip compatibility patches and addons if we can find better matches
		isCompatPatch := strings.Contains(titleLower, "compat") ||
			strings.Contains(titleLower, "patch") ||
			strings.Contains(slugLower, "compat") ||
			strings.Contains(slugLower, "patch") ||
			strings.Contains(titleLower, "trash") ||
			strings.Contains(slugLower, "trash")

		// Check for exact slug match
		if strings.EqualFold(hit.Slug, rpName) {
			rpCache[cacheKey] = hit.ProjectID
			saveRPCache(cachePath, rpCache)
			return hit.ProjectID, nil
		}

		// Check for exact title match
		if strings.EqualFold(hit.Title, rpName) {
			rpCache[cacheKey] = hit.ProjectID
			saveRPCache(cachePath, rpCache)
			return hit.ProjectID, nil
		}

		// More flexible matching: check if key search terms appear in the result
		searchTerms := strings.Fields(strings.ToLower(searchQuery))

		matches := 0
		for _, term := range searchTerms {
			if strings.Contains(titleLower, term) || strings.Contains(slugLower, term) {
				matches++
			}
		}

		minMatches := len(searchTerms) * 3 / 5
		if minMatches == 0 && len(searchTerms) > 0 {
			minMatches = 1
		}

		// If at least 60% of search terms match, consider it a good match
		// Prefer non-compatibility results
		if matches >= minMatches && len(searchTerms) > 0 {
			// If this is a compatibility patch, only use it if no better options found later
			if !isCompatPatch {
				rpCache[cacheKey] = hit.ProjectID
				saveRPCache(cachePath, rpCache)
				return hit.ProjectID, nil
			}
		}
	}

	// If we only found compatibility patches, use the first one as fallback
	for _, hit := range result.Hits {
		titleLower := strings.ToLower(hit.Title)
		slugLower := strings.ToLower(hit.Slug)
		searchTerms := strings.Fields(strings.ToLower(searchQuery))

		matches := 0
		for _, term := range searchTerms {
			if strings.Contains(titleLower, term) || strings.Contains(slugLower, term) {
				matches++
			}
		}

		minMatches := len(searchTerms) * 3 / 5
		if minMatches == 0 && len(searchTerms) > 0 {
			minMatches = 1
		}

		if matches >= minMatches && len(searchTerms) > 0 {
			rpCache[cacheKey] = hit.ProjectID
			saveRPCache(cachePath, rpCache)
			return hit.ProjectID, nil
		}
	}

	// Cache empty result
	rpCache[cacheKey] = ""
	saveRPCache(cachePath, rpCache)
	return "", fmt.Errorf("resource pack not found on Modrinth")
}

// saveRPCache saves the resource pack cache to disk
func saveRPCache(cachePath string, rpCache map[string]string) {
	if cacheData, err := json.MarshalIndent(rpCache, "", "  "); err == nil {
		os.MkdirAll(filepath.Dir(cachePath), 0755)
		os.WriteFile(cachePath, cacheData, 0644)
	}
}
