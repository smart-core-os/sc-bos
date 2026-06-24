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

// buildVersion is injected at build time
//
//	-ldflags "-X github.com/smart-core-os/sc-bos/pkg/app.buildVersion=<version>"
var buildVersion string

func init() {
	Version = VersionInfo{}
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		Version.BuildInfo = buildInfo
	}
}

type VersionInfo struct {
	*debug.BuildInfo
}

// EffectiveVersion returns the resolved version of BOS. This is suitable for reporting the running BOS version to
// e.g. the cloud update system.
//
// In order of highest to lowest priority, it returns:
//   - The value of the BOS_VERSION_OVERRIDE environment variable
//   - The value of link-time variable github.com/smart-core-os/sc-bos/pkg/app.buildVersion
//
// It returns "" when neither is set. It deliberately does not fall back to the main Go module version,
// which is "(devel)" for an unstamped build - a value the update system must never Commit as the running
// version, since it can never match a cloud artefact's version.
func EffectiveVersion() string {
	if override := os.Getenv("BOS_VERSION_OVERRIDE"); override != "" {
		return override
	}
	return buildVersion
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
