package goproto

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/tools/txtar"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestDetermineGeneratorsFromDescriptor(t *testing.T) {
	tests := []struct {
		name    string
		desc    *descriptorpb.FileDescriptorProto
		want    Generator
		wantErr bool
	}{
		{
			name: "basic proto without services",
			desc: &descriptorpb.FileDescriptorProto{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
				MessageType: []*descriptorpb.DescriptorProto{
					{Name: proto.String("Message")},
				},
			},
			want: 0,
		},
		{
			name: "service with routed API",
			desc: &descriptorpb.FileDescriptorProto{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("GetRequest"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   proto.String("name"),
								Number: proto.Int32(1),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							},
						},
					},
					{
						Name: proto.String("UpdateRequest"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   proto.String("name"),
								Number: proto.Int32(1),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							},
						},
					},
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:      proto.String("Get"),
								InputType: proto.String(".test.GetRequest"),
							},
							{
								Name:      proto.String("Update"),
								InputType: proto.String(".test.UpdateRequest"),
							},
						},
					},
				},
			},
			want: GenRouter | GenWrapper,
		},
		{
			name: "service with routed API name at different position",
			desc: &descriptorpb.FileDescriptorProto{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("GetRequest"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   proto.String("id"),
								Number: proto.Int32(1),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							},
							{
								Name:   proto.String("name"),
								Number: proto.Int32(2),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							},
						},
					},
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:      proto.String("Get"),
								InputType: proto.String(".test.GetRequest"),
							},
						},
					},
				},
			},
			want: GenRouter | GenWrapper,
		},
		{
			name: "service without routed API",
			desc: &descriptorpb.FileDescriptorProto{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("GetRequest"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   proto.String("key"),
								Number: proto.Int32(1),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							},
						},
					},
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:      proto.String("Get"),
								InputType: proto.String(".test.GetRequest"),
							},
						},
					},
				},
			},
			want: GenWrapper,
		},
		{
			name: "service with mixed request types",
			desc: &descriptorpb.FileDescriptorProto{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("GetRequest"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   proto.String("name"),
								Number: proto.Int32(1),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							},
						},
					},
					{
						Name: proto.String("ListRequest"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   proto.String("parent"),
								Number: proto.Int32(1),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							},
						},
					},
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:      proto.String("Get"),
								InputType: proto.String(".test.GetRequest"),
							},
							{
								Name:      proto.String("List"),
								InputType: proto.String(".test.ListRequest"),
							},
						},
					},
				},
			},
			want: GenWrapper,
		},
		{
			name: "service with external request types",
			desc: &descriptorpb.FileDescriptorProto{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:      proto.String("Get"),
								InputType: proto.String(".external.GetRequest"),
							},
						},
					},
				},
			},
			want: GenWrapper,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineGeneratorsFromDescriptor(tt.desc)
			if got != tt.want {
				t.Errorf("determineGeneratorsFromDescriptor() = %v (%s), want %v (%s)", got, got, tt.want, tt.want)
			}
		})
	}
}

// TestDetermineGenerators tests the file-based wrapper (integration test).
// This is slower as it requires actual file I/O and protoc execution.
func TestDetermineGenerators(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not installed")
	}

	tests := []struct {
		name    string
		txtar   string
		want    Generator
		wantErr bool
	}{
		{
			name:  "service with routed API and imports",
			txtar: "service_routed.txtar",
			want:  GenRouter | GenWrapper,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load the txtar archive
			archive := loadTxtar(t, tt.txtar)

			// Create a temporary directory
			tmpDir := t.TempDir()

			// Extract the proto file
			protoFile := extractProtoFile(t, archive, tmpDir)

			// Get the relative path from tmpDir to protoFile
			relPath, err := filepath.Rel(tmpDir, protoFile)
			if err != nil {
				t.Fatalf("getting relative path: %v", err)
			}

			// Test determineGenerators
			got, err := determineGenerators(tmpDir, relPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("determineGenerators() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("determineGenerators() = %v (%s), want %v (%s)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestAnalyzeProtoFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not installed")
	}

	tests := []struct {
		name    string
		txtar   string
		want    map[string]Generator
		wantErr bool
	}{
		{
			name:  "directory with multiple proto types",
			txtar: "multi_proto_directory.txtar",
			want: map[string]Generator{
				"basic.proto":   0,
				"routed.proto":  GenRouter | GenWrapper,
				"wrapper.proto": GenWrapper,
			},
		},
		{
			name:  "nested directories",
			txtar: "nested_directories.txtar",
			want: map[string]Generator{
				"messages.proto":        0,
				"services/routed.proto": GenRouter | GenWrapper,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load the txtar archive
			archive := loadTxtar(t, tt.txtar)

			// Create a temporary directory
			tmpDir := t.TempDir()

			// Extract all files
			extractAllFiles(t, archive, tmpDir)

			// Test analyzeProtoFiles
			got, err := analyzeProtoFiles(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("analyzeProtoFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Compare results
			if len(got) != len(tt.want) {
				t.Errorf("analyzeProtoFiles() returned %d files, want %d", len(got), len(tt.want))
			}

			for file, wantGen := range tt.want {
				gotGen, ok := got[file]
				if !ok {
					t.Errorf("analyzeProtoFiles() missing file %s", file)
					continue
				}
				if gotGen != wantGen {
					t.Errorf("analyzeProtoFiles()[%s] = %v (%s), want %v (%s)", file, gotGen, gotGen, wantGen, wantGen)
				}
			}
		})
	}
}

func TestHasNameField(t *testing.T) {
	tests := []struct {
		name string
		msg  *descriptorpb.DescriptorProto
		want bool
	}{
		{
			name: "message with string name at position 1",
			msg: &descriptorpb.DescriptorProto{
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("name"),
						Number: proto.Int32(1),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
			want: true,
		},
		{
			name: "message with name at different position",
			msg: &descriptorpb.DescriptorProto{
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("other"),
						Number: proto.Int32(1),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:   proto.String("name"),
						Number: proto.Int32(2),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
			want: true,
		},
		{
			name: "message with wrong type",
			msg: &descriptorpb.DescriptorProto{
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("name"),
						Number: proto.Int32(1),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
					},
				},
			},
			want: false,
		},
		{
			name: "message without name field",
			msg: &descriptorpb.DescriptorProto{
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("id"),
						Number: proto.Int32(1),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
			want: false,
		},
		{
			name: "empty message",
			msg: &descriptorpb.DescriptorProto{
				Field: []*descriptorpb.FieldDescriptorProto{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasNameField(tt.msg); got != tt.want {
				t.Errorf("hasNameField() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions

func loadTxtar(t *testing.T, name string) *txtar.Archive {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading txtar file: %v", err)
	}
	return txtar.Parse(data)
}

// extractProtoFile extracts to dir all files in archive, returning the absolute path the last .proto file found.
func extractProtoFile(t *testing.T, archive *txtar.Archive, dir string) string {
	t.Helper()
	var protoFile string

	// Extract all files (including proto.mod)
	for _, file := range archive.Files {
		path := filepath.Join(dir, file.Name)
		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("creating directory: %v", err)
		}
		if err := os.WriteFile(path, file.Data, 0644); err != nil {
			t.Fatalf("writing file %s: %v", file.Name, err)
		}

		// Track the proto file
		if filepath.Ext(file.Name) == ".proto" {
			protoFile = path
		}
	}

	if protoFile == "" {
		t.Fatal("no proto file found in archive")
	}
	return protoFile
}

func extractAllFiles(t *testing.T, archive *txtar.Archive, dir string) {
	t.Helper()
	for _, file := range archive.Files {
		path := filepath.Join(dir, file.Name)
		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("creating directory: %v", err)
		}
		if err := os.WriteFile(path, file.Data, 0644); err != nil {
			t.Fatalf("writing file %s: %v", file.Name, err)
		}
	}
}
