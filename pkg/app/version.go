package app

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"runtime/debug"
)

var Version VersionInfo

func init() {
	Version = VersionInfo{}
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		Version.BuildInfo = buildInfo
	}
}

type VersionInfo struct {
	*debug.BuildInfo
}

// EffectiveVersion returns the version string BOS reports to the Supervisor (via Commit) and
// compares against an update target to distinguish a successful update from a rollback.
// It is the build-info main module version, overridable for development only by the
// BOS_VERSION_OVERRIDE environment variable.
func EffectiveVersion() string {
	if override := os.Getenv("BOS_VERSION_OVERRIDE"); override != "" {
		return override
	}
	if Version.BuildInfo != nil {
		return Version.BuildInfo.Main.Version
	}
	return ""
}

func (v VersionInfo) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Accept"))
	if err != nil {
		mediaType = "text/plain"
	}

	switch mediaType {
	case "application/json", "text/json":
		w.Header().Set("Content-Type", mediaType)
		enc := json.NewEncoder(w)

		if v.BuildInfo == nil {
			enc.Encode(map[string]string{"msg": "Version Unknown"})
			return
		}
		enc.Encode(v.BuildInfo)
	case "text/plain", "*/*", "text/*":
		w.Header().Set("Content-Type", "text/plain")
		if v.BuildInfo == nil {
			fmt.Fprintf(w, "Version Unknown\n")
			return
		}

		fmt.Fprintln(w, v.BuildInfo)
	default:
		http.Error(w, fmt.Sprintf("unsupported media type: %v", mediaType), http.StatusUnsupportedMediaType)
	}
}
