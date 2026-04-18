// Package debuglog writes optional verbose HTTP traces to ~/.qmlauncher/logs/*_debug.log when enabled in launcher settings.
package debuglog

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	mu          sync.RWMutex
	on          bool
	file        *os.File
	logger      *log.Logger
	currentPath string
)

// Enabled reports whether debug file logging is active.
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return on && logger != nil
}

// SetEnabled turns debug logging on or off. When enabling, creates a new file named
// qmlauncher-gui_<timestamp>_debug.log under ~/.qmlauncher/logs.
func SetEnabled(enabled bool) error {
	mu.Lock()
	defer mu.Unlock()
	if !enabled {
		if file != nil {
			_ = file.Close()
			file = nil
			logger = nil
		}
		currentPath = ""
		on = false
		return nil
	}
	// (Re)open a fresh debug file each time debug is turned on.
	if file != nil {
		_ = file.Close()
		file = nil
		logger = nil
		currentPath = ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	logsDir := filepath.Join(home, ".qmlauncher", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}
	ts := time.Now().Format("2006-01-02_15-04-05")
	path := filepath.Join(logsDir, fmt.Sprintf("qmlauncher-gui_%s_debug.log", ts))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	if file != nil {
		_ = file.Close()
	}
	file = f
	currentPath = path
	logger = log.New(f, "", log.LstdFlags|log.Lmicroseconds)
	on = true
	logger.Printf("[debug] file=%s", path)
	return nil
}

// CurrentLogPath returns the active debug log file path, or empty if debug logging is off.
func CurrentLogPath() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentPath
}

// Printf writes one line with a [debug] prefix when logging is enabled.
func Printf(format string, args ...interface{}) {
	mu.RLock()
	lg := logger
	active := on && lg != nil
	mu.RUnlock()
	if !active {
		return
	}
	lg.Printf("[debug] "+format, args...)
}

// LogCurseForgeKeyFromQMServer logs the full key received from GET /launcher/curseforge-api-key (debug only).
// If the value looks like a bcrypt hash ($2a$/…), logs a warning — CurseForge expects the plain Core API key from Developer Console.
func LogCurseForgeKeyFromQMServer(key string) {
	if !Enabled() {
		return
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	Printf("CurseForge: QMServer cloud key (full, compare with QMAdmin / CurseForge Console): %s", key)
	if looksLikeCurseForgeDollar2Key(key) {
		// Some Core API keys legitimately start with $2a$ / similar; not necessarily "wrong" vs plain text.
		Printf("CurseForge: NOTE — key starts with $2… (known format for some Core API keys). Paste exactly as in console; any edit breaks the key. If 403 persists, regenerate or check key status in https://console.curseforge.com/")
	}
}

func looksLikeCurseForgeDollar2Key(s string) bool {
	if len(s) < 20 || !strings.HasPrefix(s, "$2") {
		return false
	}
	return strings.Count(s, "$") >= 3
}

// RedactURL returns the URL string with sensitive query parameters masked.
func RedactURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	q := u.Query()
	for _, key := range []string{"token", "access_token", "refresh_token", "password", "code"} {
		if q.Get(key) != "" {
			q.Set(key, "<redacted>")
		}
	}
	out := *u
	out.RawQuery = q.Encode()
	return out.String()
}

// RedactHeaderValue masks a single header value for logs.
func RedactHeaderValue(name, value string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case "authorization", "cookie", "x-api-key", "api-key":
		v := strings.TrimSpace(value)
		if v == "" {
			return ""
		}
		if n == "authorization" && strings.HasPrefix(strings.ToLower(v), "bearer ") {
			return "Bearer <redacted>"
		}
		if len(v) <= 8 {
			return "<redacted>"
		}
		return v[:4] + "…" + v[len(v)-2:] + " (len=" + fmt.Sprintf("%d", len(v)) + ")"
	default:
		return value
	}
}
