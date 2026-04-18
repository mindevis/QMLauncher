package network

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// UploadLocalSkinsToQMServer uploads all skins, capes and elytras from CustomSkinLoader/LocalSkin/
// to QMServer before sync, so they get distributed to other users.
// Requires Cloud account token. Logs via log.Printf.
func UploadLocalSkinsToQMServer(instanceDir, qmHost string, qmPort int, bearerToken string, logFn func(string)) error {
	if qmHost == "" || bearerToken == "" {
		return nil
	}
	if logFn == nil {
		logFn = func(s string) { log.Print(s) }
	}

	scheme := "http"
	if qmPort == 443 {
		scheme = "https"
	}
	apiBase := fmt.Sprintf("%s://%s:%d/api/v1", scheme, qmHost, qmPort)

	uploaded := 0
	for _, subdir := range []string{"skins", "capes", "elytras"} {
		dir := filepath.Join(instanceDir, "CustomSkinLoader", "LocalSkin", subdir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			logFn(fmt.Sprintf("[SkinsUpload] read dir %s: %v", dir, err))
			continue
		}

		endpoint := subdir

		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".png") {
				continue
			}
			username := strings.TrimSuffix(e.Name(), ".png")
			if len(username) < 3 || len(username) > 16 {
				continue
			}

			fullPath := filepath.Join(dir, e.Name())
			f, err := os.Open(fullPath)
			if err != nil {
				logFn(fmt.Sprintf("[SkinsUpload] open %s: %v", fullPath, err))
				continue
			}

			url := fmt.Sprintf("%s/%s/%s", apiBase, endpoint, username)
			req, err := http.NewRequest("POST", url, f)
			if err != nil {
				f.Close()
				continue
			}
			req.Header.Set("Authorization", "Bearer "+bearerToken)
			req.Header.Set("User-Agent", QMServerUserAgent)
			req.Header.Set("Content-Type", "image/png")

			resp, err := QMServerHTTPClient.Do(req)
			f.Close()
			if err != nil {
				logFn(fmt.Sprintf("[SkinsUpload] upload %s: %v", e.Name(), err))
				continue
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				uploaded++
				logFn(fmt.Sprintf("[SkinsUpload] uploaded %s/%s", subdir, e.Name()))
			} else {
				logFn(fmt.Sprintf("[SkinsUpload] upload %s: HTTP %d", e.Name(), resp.StatusCode))
			}
		}
	}

	if uploaded > 0 {
		logFn(fmt.Sprintf("[SkinsUpload] uploaded %d skin/cape/elytra file(s) to QMServer", uploaded))
	}
	return nil
}
