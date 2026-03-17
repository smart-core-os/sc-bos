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
	"sync"
)

// V0ToV1 returns the V1 service name for the given service name.
// Both versions are fully qualified names ([package].[service]).
// See the package documentation for format details.
// The returned service name will be unchanged if it is not a recognized V0 package.
func V0ToV1(fqn string) string {
	if v1, ok := v0ToV1SpecialCase(fqn); ok {
		return v1
	}
	pkg, service := splitPackageService(fqn)
	if !isBOSPackage(pkg) {
		return fqn
	}
	if isVersioned(pkg) {
		return fqn
	}
	if pkg == "smartcore.bos" {
		resource := extractResource(service)
		return "smartcore.bos." + resource + ".v1." + service
	}

	return pkg + ".v1." + service
}

// V1ToV0 returns the V0 service name for the given service name.
// Both versions are fully qualified names ([package].[service]).
// See the package documentation for format details.
// The returned service name will be unchanged if it is not a recognized V1 package.
func V1ToV0(fqn string) string {
	if v0, ok := v1ToV0SpecialCase(fqn); ok {
		return v0
	}
	pkg, service := splitPackageService(fqn)
	if !isBOSPackage(pkg) {
		return fqn
	}
	if !strings.HasSuffix(pkg, ".v1") {
		return fqn
	}
	v0Pkg := strings.TrimSuffix(pkg, ".v1")

	if v0Pkg == "smartcore.bos" {
		return fqn // invalid: smartcore.bos.v1
	}

	rest := strings.TrimPrefix(v0Pkg, "smartcore.bos.")

	// If (after removing the version) the pkg looks like smartcore.bos.a.b...
	// or the bit after smartcore.bos is a known special case
	// then we don't touch it any more
	if isNestedPackage(rest) {
		return v0Pkg + "." + service
	}
	return "smartcore.bos." + service
}

func splitPackageService(fqn string) (pkg, service string) {
	lastDot := strings.LastIndex(fqn, ".")
	if lastDot == -1 {
		return "", fqn
	}
	return fqn[:lastDot], fqn[lastDot+1:]
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

var (
	specialCases = []struct{ v0, v1 string }{
		{"smartcore.bos.tenants.TenantApi", "smartcore.bos.tenant.v1.TenantApi"},
		{"smartcore.bos.EnterLeaveHistory", "smartcore.bos.enterleavesensor.v1.EnterLeaveSensorHistory"},
	}
	specialCaseV0ToV1      map[string]string
	buildSpecialCaseV0ToV1 = sync.OnceFunc(func() {
		specialCaseV0ToV1 = make(map[string]string)
		for _, sc := range specialCases {
			specialCaseV0ToV1[sc.v0] = sc.v1
		}
	})
	specialCaseV1ToV0      map[string]string
	buildSpecialCaseV1ToV0 = sync.OnceFunc(func() {
		specialCaseV1ToV0 = make(map[string]string)
		for _, sc := range specialCases {
			specialCaseV1ToV0[sc.v1] = sc.v0
		}
	})

	// traitSpecialCases handles sc-api smartcore.traits service renames.
	// These cover typos or other cases where the service name should change during migration.
	traitSpecialCases = []struct{ v0, v1 string }{
		// MotionSensorSensorInfo has a doubled "Sensor" word — corrected to MotionSensorInfo.
		{"smartcore.traits.MotionSensorSensorInfo", "smartcore.bos.motionsensor.v1.MotionSensorInfo"},
	}
	traitSpecialCaseToV1      map[string]string
	buildTraitSpecialCaseToV1 = sync.OnceFunc(func() {
		traitSpecialCaseToV1 = make(map[string]string)
		for _, sc := range traitSpecialCases {
			traitSpecialCaseToV1[sc.v0] = sc.v1
		}
	})
	traitSpecialCaseToTraits      map[string]string
	buildTraitSpecialCaseToTraits = sync.OnceFunc(func() {
		traitSpecialCaseToTraits = make(map[string]string)
		for _, sc := range traitSpecialCases {
			traitSpecialCaseToTraits[sc.v1] = sc.v0
		}
	})
)

func v0ToV1SpecialCase(fqn string) (string, bool) {
	buildSpecialCaseV0ToV1()
	v1, ok := specialCaseV0ToV1[fqn]
	return v1, ok
}

func v1ToV0SpecialCase(fqn string) (string, bool) {
	buildSpecialCaseV1ToV0()
	v0, ok := specialCaseV1ToV0[fqn]
	return v0, ok
}

// TraitsToV1 converts a fully-qualified service name from smartcore.traits to
// smartcore.bos.{resource}.v1.  Returns fqn unchanged for non-traits packages.
// e.g. "smartcore.traits.MeterApi" -> "smartcore.bos.meter.v1.MeterApi"
func TraitsToV1(fqn string) string {
	buildTraitSpecialCaseToV1()
	if v1, ok := traitSpecialCaseToV1[fqn]; ok {
		return v1
	}
	pkg, service := splitPackageService(fqn)
	if pkg != "smartcore.traits" {
		return fqn
	}
	resource := extractResource(service)
	return "smartcore.bos." + resource + ".v1." + service
}

// V1ToTraits is the inverse of TraitsToV1: smartcore.bos.{resource}.v1.Svc -> smartcore.traits.Svc.
// Used by genrenames to detect service renames during the traits->v1 migration.
// Returns fqn unchanged for packages that don't match the expected pattern.
func V1ToTraits(fqn string) string {
	buildTraitSpecialCaseToTraits()
	if v0, ok := traitSpecialCaseToTraits[fqn]; ok {
		return v0
	}
	pkg, service := splitPackageService(fqn)
	if !isBOSPackage(pkg) {
		return fqn
	}
	if !strings.HasSuffix(pkg, ".v1") {
		return fqn
	}
	v0Pkg := strings.TrimSuffix(pkg, ".v1")
	rest := strings.TrimPrefix(v0Pkg, "smartcore.bos.")
	// Nested packages (e.g. driver.dali) are not traits.
	if isNestedPackage(rest) {
		return fqn
	}
	return "smartcore.traits." + service
}
