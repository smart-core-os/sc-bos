package protopkg

import (
	"strings"

	"github.com/smart-core-os/sc-bos/internal/compat/protopkg"
)

// newToOld converts new-style paths to old-style paths.
// Example: /smartcore.bos.meter.v1.MeterApi/GetMeterReading -> /smartcore.bos.MeterApi/GetMeterReading
func newToOld(path string) string {
	service, method, ok := parsePath(path)
	if !ok {
		return path
	}

	newService := protopkg.V1ToV0(service)
	return buildPath(newService, method)
}

// oldToNew converts old-style paths to new-style paths.
// Example: /smartcore.bos.MeterApi/GetMeterReading -> /smartcore.bos.meter.v1.MeterApi/GetMeterReading
// Example: /smartcore.traits.LightApi/GetBrightness -> /smartcore.bos.light.v1.LightApi/GetBrightness
func oldToNew(path string) string {
	service, method, ok := parsePath(path)
	if !ok {
		return path
	}

	newService := protopkg.V0ToV1(service)
	if newService == service {
		// V0ToV1 didn't recognise it; try the traits → bos migration
		newService = protopkg.TraitsToV1(service)
	}
	return buildPath(newService, method)
}

// parsePath splits a gRPC path into package, service name, and method.
// Example: /smartcore.bos.MeterApi/GetMeterReading -> (smartcore.bos.MeterApi, GetMeterReading, true)
func parsePath(path string) (service, method string, ok bool) {
	if !strings.HasPrefix(path, "/") {
		return "", "", false
	}
	return strings.Cut(path[1:], "/")
}

// buildPath constructs a gRPC path from service, and method.
func buildPath(service, method string) string {
	return "/" + service + "/" + method
}
