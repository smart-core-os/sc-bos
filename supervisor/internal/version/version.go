// Package version exposes the Supervisor's own build version.
package version

// Version is the Supervisor's version string, injected at link time via
//
//	-ldflags "-X github.com/smart-core-os/sc-bos/supervisor/internal/version.Version=<v>"
//
// by supervisor/rpm/build.sh. It is the version the Supervisor reports for itself (so a self-update
// can tell a successful upgrade from a rollback), mirroring how BOS reports its own version. It is
// empty in un-stamped builds (e.g. `go run`, `go test`, a plain `go build`).
var Version string
