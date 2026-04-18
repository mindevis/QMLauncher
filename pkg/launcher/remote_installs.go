package launcher

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"QMLauncher/internal/meta"
)

// RemoteInstallMeta describes a resource installed from CurseForge or Modrinth (catalog or launcher automation).
type RemoteInstallMeta struct {
	Category  string `json:"category"`
	Source    string `json:"source"`
	ProjectID string `json:"projectId"`
	Slug      string `json:"slug,omitempty"`
	Title     string `json:"title,omitempty"`
	// IconURL is the project thumbnail from CurseForge/Modrinth when installed from the catalog (HTTPS).
	IconURL string `json:"iconUrl,omitempty"`
}

type remoteInstallsDoc struct {
	Entries map[string]RemoteInstallMeta `json:"entries"`
}

func remoteInstallsFilePath(instanceDir string) string {
	return filepath.Join(instanceDir, ".qmlauncher", "remote-installs.json")
}

func normalizeInstallMapKeySegment(pathOrBase string) string {
	p := filepath.ToSlash(strings.TrimSpace(pathOrBase))
	parts := strings.Split(p, "/")
	if len(parts) == 0 {
		return p
	}
	last := parts[len(parts)-1]
	for strings.HasSuffix(strings.ToLower(last), ".disabled") {
		last = last[:len(last)-len(".disabled")]
		parts[len(parts)-1] = last
	}
	return strings.Join(parts, "/")
}

func installMapKey(category, resourcePathOrBase string) string {
	cat := strings.ToLower(strings.TrimSpace(category))
	base := filepath.Base(filepath.ToSlash(strings.TrimSpace(resourcePathOrBase)))
	rel := normalizeInstallMapKeySegment(base)
	if cat == "" || rel == "" {
		return ""
	}
	return cat + "/" + rel
}

// LoadRemoteInstalls reads .qmlauncher/remote-installs.json for an instance directory.
func LoadRemoteInstalls(instanceDir string) map[string]RemoteInstallMeta {
	data, err := os.ReadFile(remoteInstallsFilePath(instanceDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]RemoteInstallMeta{}
		}
		return map[string]RemoteInstallMeta{}
	}
	var doc remoteInstallsDoc
	if err := json.Unmarshal(data, &doc); err != nil || doc.Entries == nil {
		return map[string]RemoteInstallMeta{}
	}
	return doc.Entries
}

func saveRemoteInstalls(instanceDir string, entries map[string]RemoteInstallMeta) error {
	dir := filepath.Join(instanceDir, ".qmlauncher")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	doc := remoteInstallsDoc{Entries: entries}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(remoteInstallsFilePath(instanceDir), data, 0644)
}

// RecordRemoteInstall adds or updates an entry keyed by category and saved file basename.
func RecordRemoteInstall(instanceDir, category, savedBase string, meta RemoteInstallMeta) {
	entries := LoadRemoteInstalls(instanceDir)
	key := installMapKey(category, filepath.Base(strings.TrimSpace(savedBase)))
	if key == "" {
		return
	}
	meta.Category = strings.ToLower(strings.TrimSpace(category))
	meta.Source = strings.ToLower(strings.TrimSpace(meta.Source))
	entries[key] = meta
	_ = saveRemoteInstalls(instanceDir, entries)
}

// RemoveRemoteInstall removes the entry for a resource path (basename or rel; .disabled normalized).
func RemoveRemoteInstall(instanceDir, category, resourcePath string) {
	entries := LoadRemoteInstalls(instanceDir)
	key := installMapKey(category, resourcePath)
	if key == "" {
		return
	}
	delete(entries, key)
	_ = saveRemoteInstalls(instanceDir, entries)
}

// EnsureFabricAPIRemoteInstallRecord writes Fabric API Modrinth metadata when a fabric-api jar is present
// but was installed before we tracked remote-installs (or migrated instances).
func EnsureFabricAPIRemoteInstallRecord(instanceDir string, mods []string) {
	for _, name := range mods {
		active := strings.TrimSpace(name)
		lower := strings.ToLower(active)
		var base string
		if strings.HasSuffix(lower, ".jar.disabled") {
			base = filepath.Base(active[:len(active)-len(".disabled")])
		} else if strings.HasSuffix(lower, ".jar") {
			base = filepath.Base(active)
		} else {
			continue
		}
		if !strings.Contains(strings.ToLower(base), "fabric-api") {
			continue
		}
		key := installMapKey("mods", base)
		if key == "" {
			continue
		}
		entries := LoadRemoteInstalls(instanceDir)
		if rec, ok := entries[key]; ok && strings.EqualFold(strings.TrimSpace(rec.Slug), meta.FabricAPIModrinthSlug) {
			return
		}
		RecordRemoteInstall(instanceDir, "mods", base, RemoteInstallMeta{
			Category:  "mods",
			Source:    "modrinth",
			ProjectID: meta.FabricAPIModrinthProjectID,
			Slug:      meta.FabricAPIModrinthSlug,
			Title:     "Fabric API",
		})
		return
	}
}
