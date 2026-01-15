package protov1

import (
	"testing"
)

// TestUpdatePackageDeclaration tests edge cases not covered by integration tests
func TestUpdatePackageDeclaration(t *testing.T) {
	tests := []struct {
		name    string
		content string
		oldPkg  string
		newPkg  string
		want    string
	}{
		{
			name:    "trailing spaces",
			content: "package smartcore.bos  ;",
			oldPkg:  "smartcore.bos",
			newPkg:  "smartcore.bos.meter.v1",
			want:    "package smartcore.bos.meter.v1;",
		},
		{
			name:    "nested package - driver.dali",
			content: "package smartcore.bos.driver.dali;",
			oldPkg:  "smartcore.bos.driver.dali",
			newPkg:  "smartcore.bos.driver.dali.v1",
			want:    "package smartcore.bos.driver.dali.v1;",
		},
		{
			name:    "nested package - tenants",
			content: "package smartcore.bos.tenants;",
			oldPkg:  "smartcore.bos.tenants",
			newPkg:  "smartcore.bos.tenants.v1",
			want:    "package smartcore.bos.tenants.v1;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updatePackageDeclaration(tt.content, tt.oldPkg, tt.newPkg)
			if got != tt.want {
				t.Errorf("updatePackageDeclaration() got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestUpdateImportPaths tests edge cases not covered by integration tests
func TestUpdateImportPaths(t *testing.T) {
	// Create a protoFile that simulates a moved health.proto
	allFiles := []protoFile{
		{
			oldPath: "/repo/proto/health.proto",
			newPath: "/repo/proto/smartcore/bos/health/v1/health.proto",
			baseDir: "/repo/proto",
		},
	}

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "preserves leading spaces",
			content: `  import "health.proto";`,
			want:    `  import "smartcore/bos/health/v1/health.proto";`,
		},
		{
			name:    "already versioned import unchanged",
			content: `import "smartcore/bos/health/v1/health.proto";`,
			want:    `import "smartcore/bos/health/v1/health.proto";`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateImportPaths(tt.content, allFiles)
			if got != tt.want {
				t.Errorf("updateImportPaths() %s:\ngot:  %q\nwant: %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestExtractTypeNames tests that we correctly extract message and enum names from proto content
func TestExtractTypeNames(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name: "top-level types only",
			content: `syntax = "proto3";
message Actor {
  string name = 1;
}
enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
}
message AccessAttempt {
  string id = 1;
}`,
			want: []string{"Actor", "AccessAttempt", "Status"},
		},
		{
			name: "excludes nested types",
			content: `syntax = "proto3";
message EmergencyStatus {
  enum Test {
    TEST_UNKNOWN = 0;
    FUNCTION_TEST = 2;
  }
  enum Mode {
    MODE_UNSPECIFIED = 0;
    REST = 1;
  }
  message Details {
    string info = 1;
  }
  repeated Test pending_tests = 1;
}
message TopLevel {
  string id = 1;
}`,
			want: []string{"EmergencyStatus", "TopLevel"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTypeNames([]byte(tt.content))
			if len(got) != len(tt.want) {
				t.Errorf("extractTypeNames() got %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractTypeNames() got %v, want %v", got, tt.want)
					return
				}
			}
		})
	}
}

// TestUpdateTypeReferences tests that type references are properly qualified
func TestUpdateTypeReferences(t *testing.T) {
	typeToPackage := map[string]string{
		"Actor":         "smartcore.bos.actor.v1",
		"BaseType":      "smartcore.bos.base.v1",
		"AccessAttempt": "smartcore.bos.access.v1",
		"HealthCheck":   "smartcore.bos.health.v1",
	}

	tests := []struct {
		name           string
		content        string
		currentPackage string
		want           string
	}{
		{
			name: "cross-package qualification and same-package preservation",
			content: `message AccessAttempt {
  Actor actor = 1;
  repeated Actor actors = 2;
}
message HistoryRecord {
  BaseType value = 1;
}`,
			currentPackage: "smartcore.bos.base.v1",
			want: `message AccessAttempt {
  smartcore.bos.actor.v1.Actor actor = 1;
  repeated smartcore.bos.actor.v1.Actor actors = 2;
}
message HistoryRecord {
  BaseType value = 1;
}`,
		},
		{
			name: "built-in and already qualified types unchanged",
			content: `message Test {
  string name = 1;
  int32 count = 2;
  bool active = 3;
  google.protobuf.Timestamp time = 4;
  ExternalType ext = 5;
}`,
			currentPackage: "smartcore.bos.test.v1",
			want: `message Test {
  string name = 1;
  int32 count = 2;
  bool active = 3;
  google.protobuf.Timestamp time = 4;
  ExternalType ext = 5;
}`,
		},
		{
			name: "nested types with cross-package and same-package references",
			content: `message PullAccessAttemptsResponse {
  repeated Change changes = 1;

  message Change {
    string name = 1;
    AccessAttempt access_attempt = 2;
    Actor actor = 3;
  }
  
  enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
  }
}

message ServiceTicket {
  message Classification {
    string category = 1;
    Actor assigned_to = 2;
  }
  
  Classification classification = 1;
  AccessAttempt related_access = 2;
}`,
			currentPackage: "smartcore.bos.access.v1",
			want: `message PullAccessAttemptsResponse {
  repeated Change changes = 1;

  message Change {
    string name = 1;
    AccessAttempt access_attempt = 2;
    smartcore.bos.actor.v1.Actor actor = 3;
  }
  
  enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
  }
}

message ServiceTicket {
  message Classification {
    string category = 1;
    smartcore.bos.actor.v1.Actor assigned_to = 2;
  }
  
  Classification classification = 1;
  AccessAttempt related_access = 2;
}`,
		},
		{
			name: "optional fields with cross-package types",
			content: `message Allocation {
  string id = 1;
  int32 assignment = 2;
  optional Actor actor = 3;
  optional smartcore.types.time.Period period = 4;
}

message OtherMessage {
  optional BaseType value = 1;
  optional AccessAttempt attempt = 2;
}`,
			currentPackage: "smartcore.bos.allocation.v1",
			want: `message Allocation {
  string id = 1;
  int32 assignment = 2;
  optional smartcore.bos.actor.v1.Actor actor = 3;
  optional smartcore.types.time.Period period = 4;
}

message OtherMessage {
  optional smartcore.bos.base.v1.BaseType value = 1;
  optional smartcore.bos.access.v1.AccessAttempt attempt = 2;
}`,
		},
		{
			name: "old-style smartcore.bos qualified names",
			content: `message Device {
  string name = 1;
  smartcore.traits.Metadata metadata = 2;
  repeated smartcore.bos.HealthCheck health_checks = 3;
}

message Status {
  smartcore.bos.Actor actor = 1;
  smartcore.bos.BaseType base = 2;
  google.protobuf.Timestamp time = 3;
}`,
			currentPackage: "smartcore.bos.devices.v1",
			want: `message Device {
  string name = 1;
  smartcore.traits.Metadata metadata = 2;
  repeated smartcore.bos.health.v1.HealthCheck health_checks = 3;
}

message Status {
  smartcore.bos.actor.v1.Actor actor = 1;
  smartcore.bos.base.v1.BaseType base = 2;
  google.protobuf.Timestamp time = 3;
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateTypeReferences(tt.content, tt.currentPackage, typeToPackage)
			if got != tt.want {
				t.Errorf("updateTypeReferences() %s:\ngot:\n%s\nwant:\n%s", tt.name, got, tt.want)
			}
		})
	}
}

// TestUpdateServiceDeclarations tests renaming of service declarations
func TestUpdateServiceDeclarations(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		serviceRenames map[string]string
		want           string
	}{
		{
			name: "single service rename",
			content: `syntax = "proto3";

service EnterLeaveHistory {
  rpc ListHistory(Request) returns (Response);
}`,
			serviceRenames: map[string]string{
				"EnterLeaveHistory": "EnterLeaveSensorHistory",
			},
			want: `syntax = "proto3";

service EnterLeaveSensorHistory {
  rpc ListHistory(Request) returns (Response);
}`,
		},
		{
			name: "multiple services - all renamed",
			content: `syntax = "proto3";

service EnterLeaveHistory {
  rpc ListHistory(Request) returns (Response);
}

service EnterLeaveInfo {
  rpc GetInfo(Request) returns (Response);
}`,
			serviceRenames: map[string]string{
				"EnterLeaveHistory": "EnterLeaveSensorHistory",
				"EnterLeaveInfo":    "EnterLeaveSensorInfo",
			},
			want: `syntax = "proto3";

service EnterLeaveSensorHistory {
  rpc ListHistory(Request) returns (Response);
}

service EnterLeaveSensorInfo {
  rpc GetInfo(Request) returns (Response);
}`,
		},
		{
			name: "multiple services - partial rename",
			content: `syntax = "proto3";

service TenantApi {
  rpc GetTenant(Request) returns (Response);
}

service MeterApi {
  rpc GetMeter(Request) returns (Response);
}`,
			serviceRenames: map[string]string{
				"TenantApi": "TenantServiceApi", // Only this one gets renamed
				// MeterApi stays the same
			},
			want: `syntax = "proto3";

service TenantServiceApi {
  rpc GetTenant(Request) returns (Response);
}

service MeterApi {
  rpc GetMeter(Request) returns (Response);
}`,
		},
		{
			name: "no renames - empty map",
			content: `syntax = "proto3";

service MeterApi {
  rpc GetMeter(Request) returns (Response);
}`,
			serviceRenames: map[string]string{},
			want: `syntax = "proto3";

service MeterApi {
  rpc GetMeter(Request) returns (Response);
}`,
		},
		{
			name: "no renames - nil map",
			content: `syntax = "proto3";

service AlertApi {
  rpc GetAlert(Request) returns (Response);
}`,
			serviceRenames: nil,
			want: `syntax = "proto3";

service AlertApi {
  rpc GetAlert(Request) returns (Response);
}`,
		},
		{
			name: "service with extra spaces in declaration",
			content: `syntax = "proto3";

service  EnterLeaveHistory  {
  rpc ListHistory(Request) returns (Response);
}`,
			serviceRenames: map[string]string{
				"EnterLeaveHistory": "EnterLeaveSensorHistory",
			},
			want: `syntax = "proto3";

service EnterLeaveSensorHistory {
  rpc ListHistory(Request) returns (Response);
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateServiceDeclarations(tt.content, tt.serviceRenames)
			if got != tt.want {
				t.Errorf("updateServiceDeclarations() %s:\ngot:\n%s\nwant:\n%s", tt.name, got, tt.want)
			}
		})
	}
}
