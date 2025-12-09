package protopkg

import (
	"strings"
)

// newToOld converts new-style paths to old-style paths.
// Example: /smartcore.bos.meter.v1.MeterApi/GetMeterReading -> /smartcore.bos.MeterApi/GetMeterReading
func newToOld(path string) string {
	servicePath, method, ok := parsePath(path)
	if !ok {
		return path
	}

	service := lastSegment(servicePath)
	return buildPath("smartcore.bos", service, method)
}

// oldToNew converts old-style paths to new-style paths.
// Example: /smartcore.bos.MeterApi/GetMeterReading -> /smartcore.bos.meter.v1.MeterApi/GetMeterReading
func oldToNew(path string) string {
	servicePath, method, ok := parsePath(path)
	if !ok {
		return path
	}

	service := lastSegment(servicePath)
	resource := extractResource(service)
	return buildPath("smartcore.bos."+resource+".v1", service, method)
}

// parsePath splits a gRPC path into service path and method.
// Example: /smartcore.bos.MeterApi/GetMeterReading -> (smartcore.bos.MeterApi, GetMeterReading, true)
func parsePath(path string) (servicePath, method string, ok bool) {
	if !strings.HasPrefix(path, "/smartcore.bos.") {
		return "", "", false
	}

	parts := strings.SplitN(path[1:], "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
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

// extractResource derives the resource name from a service name.
// Examples:
//   - MeterApi -> meter
//   - MeterInfo -> meter
//   - MeterHistory -> meter
//   - AlertAdminApi -> alert
//   - ElectricHistory -> electric
func extractResource(service string) string {
	// Handle History suffix
	if strings.HasSuffix(service, "History") {
		base := strings.TrimSuffix(service, "History")
		return strings.ToLower(base)
	}

	// Handle AdminApi suffix (it's still the same resource, just admin operations)
	if strings.HasSuffix(service, "AdminApi") {
		base := strings.TrimSuffix(service, "AdminApi")
		return strings.ToLower(base)
	}

	// Handle Info suffix
	if strings.HasSuffix(service, "Info") {
		base := strings.TrimSuffix(service, "Info")
		return strings.ToLower(base)
	}

	// Handle Api suffix
	if strings.HasSuffix(service, "Api") {
		base := strings.TrimSuffix(service, "Api")
		return strings.ToLower(base)
	}

	return strings.ToLower(service)
}
