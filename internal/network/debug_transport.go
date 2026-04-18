package network

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"QMLauncher/internal/debuglog"
)

const debugMaxBodyLog = 8192

// debugCondTransport logs HTTP requests and responses when debug logging is enabled.
type debugCondTransport struct {
	inner http.RoundTripper
}

func (d *debugCondTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !debuglog.Enabled() {
		return d.inner.RoundTrip(req)
	}

	u := ""
	if req.URL != nil {
		u = debuglog.RedactURL(req.URL)
	}
	debuglog.Printf("HTTP → %s %s", req.Method, u)
	for k, vals := range req.Header {
		kn := http.CanonicalHeaderKey(k)
		for _, v := range vals {
			debuglog.Printf("HTTP   req header %s: %s", kn, debuglog.RedactHeaderValue(kn, v))
		}
	}

	var reqBody []byte
	if req.Body != nil {
		reqBody, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
	}
	if len(reqBody) > 0 {
		if req.URL != nil && hostOmitsCredentialBodies(req.URL.Host) {
			debuglog.Printf("HTTP   req body: <%d bytes omitted (oauth/auth host)>", len(reqBody))
		} else if len(reqBody) <= debugMaxBodyLog && likelyTextContent(req.Header.Get("Content-Type")) {
			debuglog.Printf("HTTP   req body (%d B): %s", len(reqBody), truncateRunes(string(reqBody), debugMaxBodyLog))
		} else if len(reqBody) > debugMaxBodyLog {
			debuglog.Printf("HTTP   req body: <%d bytes, omitted>", len(reqBody))
		}
	}

	resp, err := d.inner.RoundTrip(req)
	if err != nil {
		debuglog.Printf("HTTP ← error: %v", err)
		return resp, err
	}

	ct := resp.Header.Get("Content-Type")
	cl := resp.ContentLength
	debuglog.Printf("HTTP ← %s (%s) Content-Length=%d", resp.Status, ct, cl)

	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if readErr != nil {
		debuglog.Printf("HTTP   read body error: %v", readErr)
		return resp, nil
	}

	host := ""
	if req.URL != nil {
		host = req.URL.Host
	}
	if req.URL != nil && strings.Contains(req.URL.Path, "/launcher/curseforge-api-key") {
		debuglog.Printf("HTTP   resp body: <json omitted — full key is logged once as \"CurseForge: QMServer cloud key\" after parse>")
		return resp, nil
	}
	logRespBody(host, ct, body)
	return resp, nil
}

// hostOmitsCredentialBodies is true for hosts where request/response bodies may contain tokens (never log raw).
func hostOmitsCredentialBodies(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if i := strings.Index(h, ":"); i >= 0 {
		h = h[:i]
	}
	switch {
	case strings.HasSuffix(h, "microsoftonline.com"):
		return true
	case strings.HasSuffix(h, "xboxlive.com"):
		return true
	case strings.HasSuffix(h, "minecraftservices.com"):
		return true
	case strings.HasSuffix(h, "live.com"):
		return true
	default:
		return false
	}
}

// WrapRoundTripperWithDebug wraps an existing RoundTripper with optional HTTP tracing (launcher_debug).
func WrapRoundTripperWithDebug(inner http.RoundTripper) http.RoundTripper {
	return &debugCondTransport{inner: inner}
}

func likelyTextContent(ct string) bool {
	ct = strings.ToLower(ct)
	if ct == "" {
		return true
	}
	return strings.Contains(ct, "json") || strings.Contains(ct, "text/") ||
		strings.Contains(ct, "xml") || strings.Contains(ct, "form-urlencoded")
}

func logRespBody(host, ct string, body []byte) {
	n := len(body)
	if n == 0 {
		debuglog.Printf("HTTP   resp body: <empty>")
		return
	}
	if hostOmitsCredentialBodies(host) {
		debuglog.Printf("HTTP   resp body: <%d bytes omitted (oauth/auth host)>", n)
		return
	}
	if !likelyTextContent(ct) || !isMostlyPrintable(body) {
		debuglog.Printf("HTTP   resp body: <binary or non-text, %d bytes>", n)
		return
	}
	if n > debugMaxBodyLog {
		debuglog.Printf("HTTP   resp body (%d B, truncated): %s…", n, truncateRunes(string(body), debugMaxBodyLog))
		return
	}
	debuglog.Printf("HTTP   resp body (%d B): %s", n, string(body))
}

func isMostlyPrintable(b []byte) bool {
	n := len(b)
	if n > 1024 {
		n = 1024
	}
	if n == 0 {
		return true
	}
	ok := 0
	for i := 0; i < n; i++ {
		c := b[i]
		if c >= 32 && c < 127 || c == '\n' || c == '\r' || c == '\t' {
			ok++
		}
	}
	return ok*10 >= n*8
}

func truncateRunes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes]
}
