package meta

import "strings"

// NormalizeCurseForgeAPIKey trims and strips invisible characters often present when pasting or storing keys (BOM, zero-width).
func NormalizeCurseForgeAPIKey(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\ufeff")
	for _, z := range []string{"\u200b", "\u200c", "\u200d", "\ufeff"} {
		s = strings.ReplaceAll(s, z, "")
	}
	return strings.TrimSpace(s)
}
