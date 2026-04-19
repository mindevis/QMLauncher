package version

// Current is used by auto-update and User-Agent. Release builds MUST set the same value as main.version via:
//
//	-X QMLauncher/internal/version.Current=$(VERSION)
//
// (see Makefile / CI). This default is for plain `go build` only.
var Current = "v1.0.10"
