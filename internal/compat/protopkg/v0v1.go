// Package protopkg provides utilities for translating proto service package names between v0 and v1 formats.
//
// # Package Structure Formats
//
// V0 format:
//   - Structure: smartcore.bos or smartcore.bos.[pkg*]
//   - Example: smartcore.bos.MeterApi
//   - Example: smartcore.bos.driver.dali.DaliApi
//   - Example: smartcore.bos.tenants.TenantApi
//
// V1 format:
//   - Structure: smartcore.bos.resource.v1 or smartcore.bos.[pkg*].v1
//   - Example: smartcore.bos.meter.v1.MeterApi
//   - Example: smartcore.bos.driver.dali.v1.DaliApi
//   - Example: smartcore.bos.tenants.v1.TenantApi
//
// The resource name is derived from the service name (e.g., MeterApi -> meter, AlertAdminApi -> alert).
package protopkg

import (
	"strings"
)

// V0ToV1 returns the V1 package name for the given pkg and service.
// See the package documentation for format details.
// The returned package will be unchanged if it is not a recognized V0 package.
func V0ToV1(pkg, service string) string {
	if !isBOSPackage(pkg) {
		return pkg
	}
	if isVersioned(pkg) {
		return pkg
	}
	if pkg == "smartcore.bos" {
		resource := extractResource(service)
		return "smartcore.bos." + resource + ".v1"
	}

	return pkg + ".v1"
}

// V1ToV0 returns the V0 package name for the given pkg and service.
// See the package documentation for format details.
// The returned package will be unchanged if it is not a recognized V1 package.
func V1ToV0(pkg, service string) string {
	_ = service // unused, kept for API consistency
	if !isBOSPackage(pkg) {
		return pkg
	}
	if !strings.HasSuffix(pkg, ".v1") {
		return pkg
	}
	v0Pkg := strings.TrimSuffix(pkg, ".v1")

	if v0Pkg == "smartcore.bos" {
		return pkg // invalid: smartcore.bos.v1
	}

	rest := strings.TrimPrefix(v0Pkg, "smartcore.bos.")

	// If (after removing the version) the pkg looks like smartcore.bos.a.b...
	// or the bit after smartcore.bos is a known special case
	// then we don't touch it any more
	if isNestedPackage(rest) {
		return v0Pkg
	}
	if isKnownNamespace(rest) {
		return v0Pkg
	}
	return "smartcore.bos"
}

func isBOSPackage(pkg string) bool {
	return pkg == "smartcore.bos" || strings.HasPrefix(pkg, "smartcore.bos.")
}

func isVersioned(pkg string) bool {
	lastSeg := lastSegment(pkg)
	// matches v1 and v2.3-beta intentionally
	return len(lastSeg) > 1 && lastSeg[0] == 'v' && isDigit(lastSeg[1])
}

func isNestedPackage(segment string) bool {
	return strings.Contains(segment, ".")
}

func lastSegment(path string) string {
	lastDot := strings.LastIndex(path, ".")
	if lastDot == -1 {
		return path
	}
	return path[lastDot+1:]
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isKnownNamespace(segment string) bool {
	// Non-standard namespaces that don't follow the resource extraction pattern.
	// These have their own proto files and should not collapse to "smartcore.bos".
	knownNamespaces := []string{"tenants"}
	for _, ns := range knownNamespaces {
		if segment == ns {
			return true
		}
	}
	return false
}

// extractResource derives the resource name from a service name.
// Examples: MeterApi -> meter, MeterHistory -> meter, AlertAdminApi -> alert
func extractResource(service string) string {
	resourceSuffixes := []string{"History", "AdminApi", "Info", "Api", "Service"}

	for _, suffix := range resourceSuffixes {
		if strings.HasSuffix(service, suffix) {
			base := strings.TrimSuffix(service, suffix)
			return strings.ToLower(base)
		}
	}

	return strings.ToLower(service)
}
