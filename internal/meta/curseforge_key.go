package meta

import (
	"os"
	"strings"
	"sync"
	"time"
)

// keyChooser returns the effective CurseForge API key (after CURSEFORGE_API_KEY env), implementing
// launcher policy: force-local, Pro+QMServer key, then settings. Registered from main (QMLauncher).
var (
	keyChooserMu sync.Mutex
	keyChooser   func() string
)

// cfCloudNegativeCache avoids hammering QMServer when the launcher endpoint returns 403 / empty key.
var (
	cfResolveMu       sync.Mutex
	cfNegativeUntil   time.Time
	cfNegativeBackoff = 5 * time.Minute
)

// SetCurseForgeKeyChooser registers the function that resolves the key from settings + cloud (env is handled here).
func SetCurseForgeKeyChooser(f func() string) {
	keyChooserMu.Lock()
	defer keyChooserMu.Unlock()
	keyChooser = f
}

var curseForgeAPI403Handler func()

// RegisterCurseForgeAPI403Handler registers a callback when CurseForge returns HTTP 403 (e.g. invalidate cached QMServer cloud key).
func RegisterCurseForgeAPI403Handler(f func()) {
	curseForgeAPI403Handler = f
}

func notifyCurseForgeAPI403() {
	if curseForgeAPI403Handler != nil {
		curseForgeAPI403Handler()
	}
}

// IsCurseForgeCloudThrottled returns true while cloud CurseForge key probes are backed off after a miss.
func IsCurseForgeCloudThrottled() bool {
	cfResolveMu.Lock()
	defer cfResolveMu.Unlock()
	return time.Now().Before(cfNegativeUntil)
}

// MarkCurseForgeCloudKeyMiss records a failed cloud lookup to throttle retries.
func MarkCurseForgeCloudKeyMiss() {
	cfResolveMu.Lock()
	defer cfResolveMu.Unlock()
	cfNegativeUntil = time.Now().Add(cfNegativeBackoff)
}

// ResetCurseForgeCloudKeyMiss clears the throttle (e.g. after cloud login/logout).
func ResetCurseForgeCloudKeyMiss() {
	cfResolveMu.Lock()
	defer cfResolveMu.Unlock()
	cfNegativeUntil = time.Time{}
}

// CurseForgeAPIKey returns the CurseForge Core API key: CURSEFORGE_API_KEY env, then the registered chooser.
func CurseForgeAPIKey() string {
	if e := strings.TrimSpace(os.Getenv("CURSEFORGE_API_KEY")); e != "" {
		return NormalizeCurseForgeAPIKey(e)
	}
	keyChooserMu.Lock()
	fn := keyChooser
	keyChooserMu.Unlock()
	if fn != nil {
		return NormalizeCurseForgeAPIKey(fn())
	}
	return ""
}
