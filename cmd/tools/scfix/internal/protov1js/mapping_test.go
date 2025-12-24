package protov1js

import (
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
)

func TestBuildJSImportMapping(t *testing.T) {
	fsys := fstest.MapFS{
		"smartcore/bos/meter/v1/meter_pb.js":          {Data: []byte("// Generated JS\n")},
		"smartcore/bos/alerts/v1/alerts_pb.js":        {Data: []byte("// Generated JS\n")},
		"smartcore/bos/driver/dali/v1/dali_pb.js":     {Data: []byte("// Generated JS\n")},
		"smartcore/bos/meter/v1/meter_grpc_web_pb.js": {Data: []byte("// Generated gRPC-Web\n")},
		"types/info_pb.js":                            {Data: []byte("// Not versioned\n")},
	}

	mapping, err := buildJSImportMapping(fsys)
	if err != nil {
		t.Fatalf("buildJSImportMapping failed: %v", err)
	}

	expected := map[string]string{
		"meter_pb":          "smartcore/bos/meter/v1/meter_pb",
		"meter_grpc_web_pb": "smartcore/bos/meter/v1/meter_grpc_web_pb",
		"alerts_pb":         "smartcore/bos/alerts/v1/alerts_pb",
		"dali_pb":           "smartcore/bos/driver/dali/v1/dali_pb",
	}

	if diff := cmp.Diff(expected, mapping); diff != "" {
		t.Errorf("mapping mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildJSImportMapping_NodeModules(t *testing.T) {
	fsys := fstest.MapFS{
		"smartcore/bos/meter/v1/meter_pb.js":            {Data: []byte("// Generated JS\n")},
		"smartcore/bos/meter/v1/meter_grpc_web_pb.js":   {Data: []byte("// Generated gRPC-Web\n")},
		"smartcore/bos/alerts/v1/alerts_pb.js":          {Data: []byte("// Generated JS\n")},
		"smartcore/bos/alerts/v1/alerts_grpc_web_pb.js": {Data: []byte("// Generated gRPC-Web\n")},
	}

	mapping, err := buildJSImportMapping(fsys)
	if err != nil {
		t.Fatalf("buildJSImportMapping failed: %v", err)
	}

	expected := map[string]string{
		"meter_pb":           "smartcore/bos/meter/v1/meter_pb",
		"meter_grpc_web_pb":  "smartcore/bos/meter/v1/meter_grpc_web_pb",
		"alerts_pb":          "smartcore/bos/alerts/v1/alerts_pb",
		"alerts_grpc_web_pb": "smartcore/bos/alerts/v1/alerts_grpc_web_pb",
	}

	if diff := cmp.Diff(expected, mapping); diff != "" {
		t.Errorf("mapping mismatch (-want +got):\n%s", diff)
	}
}

func TestVersionedJSPathPattern(t *testing.T) {
	tests := []struct {
		path      string
		wantMatch bool
	}{
		{"smartcore/bos/meter/v1/meter_pb.js", true},
		{"smartcore/bos/alerts/v1/alerts_pb.js", true},
		{"smartcore/bos/driver/dali/v1/dali_pb.js", true},
		{"smartcore/bos/driver/dali/v2/dali_pb.js", true},
		{"smartcore/bos/meter/v1/meter_grpc_web_pb.js", true},
		{"types/info_pb.js", false},
		{"traits/metadata_pb.js", false},
		{"meter_pb.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			matches := versionedJSPathPattern.FindStringSubmatch(tt.path)
			gotMatch := matches != nil
			if gotMatch != tt.wantMatch {
				t.Errorf("pattern match = %v, want %v", gotMatch, tt.wantMatch)
			}
		})
	}
}
