package protopkg

import (
	"strings"

	"github.com/smart-core-os/sc-bos/internal/compat/protopkg"
)

// newToOld converts new-style paths to old-style paths.
// Example: /smartcore.bos.meter.v1.MeterApi/GetMeterReading -> /smartcore.bos.MeterApi/GetMeterReading
func newToOld(path string) string {
	pkg, service, method, ok := parsePath(path)
	if !ok {
		return path
	}

	newPkg := protopkg.V1ToV0(pkg, service)
	if newPkg == pkg {
		return path
	}
	return buildPath(newPkg, service, method)
}

// oldToNew converts old-style paths to new-style paths.
// Example: /smartcore.bos.MeterApi/GetMeterReading -> /smartcore.bos.meter.v1.MeterApi/GetMeterReading
func oldToNew(path string) string {
	pkg, service, method, ok := parsePath(path)
	if !ok {
		return path
	}

	newPkg := protopkg.V0ToV1(pkg, service)
	if newPkg == pkg {
		return path
	}
	return buildPath(newPkg, service, method)
}

// parsePath splits a gRPC path into package, service name, and method.
// Example: /smartcore.bos.MeterApi/GetMeterReading -> (smartcore.bos, MeterApi, GetMeterReading, true)
func parsePath(path string) (pkg, service, method string, ok bool) {
	if !strings.HasPrefix(path, "/") {
		return "", "", "", false
	}

	fullService, method, ok := strings.Cut(path[1:], "/")
	if !ok {
		return "", "", "", false
	}

	service = lastSegment(fullService)
	if service == fullService {
		return "", "", "", false
	}

	pkg = fullService[:len(fullService)-len(service)-1]
	return pkg, service, method, true
}

// lastSegment returns the last dot-separated segment of a path.
// Example: smartcore.bos.meter.v1.MeterApi -> MeterApi
func lastSegment(path string) string {
	lastDot := strings.LastIndex(path, ".")
	if lastDot == -1 {
		return path
	}
	return path[lastDot+1:]
}

// buildPath constructs a gRPC path from package, service, and method.
func buildPath(pkg, service, method string) string {
	return "/" + pkg + "." + service + "/" + method
}
