// Package protopkg provides utilities for translating proto package names between version formats.
//
// # Package Formats
//
//	V0 format: smartcore.bos.SomeTraitApi
//	V1 format: smartcore.bos.sometrait.v1.SomeTraitApi
package protopkg

import (
	"strings"
)

// V0ToV1 converts a v0 package and service name to v1 format.
// Returns the v1 package name. The service name remains unchanged.
//
// Example: ("smartcore.bos", "MeterApi") -> "smartcore.bos.meter.v1"
func V0ToV1(pkg, service string) string {
	if pkg != "smartcore.bos" {
		return pkg
	}
	resource := extractResource(service)
	return "smartcore.bos." + resource + ".v1"
}

// V1ToV0 converts a v1 package name to v0 format.
// Returns the v0 package name. The service name remains unchanged.
//
// Example: ("smartcore.bos.meter.v1", "MeterApi") -> "smartcore.bos"
func V1ToV0(pkg, service string) string {
	_ = service // unused, kept for API consistency
	if !strings.HasPrefix(pkg, "smartcore.bos.") {
		return pkg
	}
	// Remove the "smartcore.bos." prefix to check the rest
	rest := strings.TrimPrefix(pkg, "smartcore.bos.")
	// Count dots - should be exactly 1 (resource.v1)
	if strings.Count(rest, ".") != 1 {
		return pkg
	}
	if lastSegment(pkg) != "v1" {
		return pkg
	}
	return "smartcore.bos"
}

func lastSegment(path string) string {
	lastDot := strings.LastIndex(path, ".")
	if lastDot == -1 {
		return path
	}
	return path[lastDot+1:]
}

// extractResource derives the resource name from a service name.
// Examples: MeterApi -> meter, MeterHistory -> meter, AlertAdminApi -> alert
func extractResource(service string) string {
	if strings.HasSuffix(service, "History") {
		base := strings.TrimSuffix(service, "History")
		return strings.ToLower(base)
	}

	if strings.HasSuffix(service, "AdminApi") {
		base := strings.TrimSuffix(service, "AdminApi")
		return strings.ToLower(base)
	}

	if strings.HasSuffix(service, "Info") {
		base := strings.TrimSuffix(service, "Info")
		return strings.ToLower(base)
	}

	if strings.HasSuffix(service, "Api") {
		base := strings.TrimSuffix(service, "Api")
		return strings.ToLower(base)
	}

	return strings.ToLower(service)
}
