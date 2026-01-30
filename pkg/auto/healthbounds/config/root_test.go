package config

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/internal/protobuf/protopath2"
)

func TestValue_Parse(t *testing.T) {
	md := (&traits.Metadata{}).ProtoReflect().Descriptor()
	mustParsePath := func(md protoreflect.MessageDescriptor, s string) protopath.Path {
		p, err := protopath2.ParsePath(md, s)
		if err != nil {
			t.Fatalf("failed to parse path %q: %v", s, err)
		}
		return p
	}
	newFieldMask := func(paths ...string) *fieldmaskpb.FieldMask {
		return &fieldmaskpb.FieldMask{Paths: paths}
	}

	tests := []struct {
		name     string
		v        Value
		md       protoreflect.MessageDescriptor
		wantPath protopath.Path
		wantFM   *fieldmaskpb.FieldMask
		wantErr  bool
	}{
		{
			name:     "empty",
			v:        "",
			md:       md,
			wantPath: mustParsePath(md, ""),
			wantFM:   nil,
		},
		{
			name:     "proto fields",
			v:        "product.firmware_version",
			md:       md,
			wantPath: mustParsePath(md, "product.firmware_version"),
			wantFM:   newFieldMask("product.firmware_version"),
		},
		{
			name:     "json fields",
			v:        "product.firmwareVersion",
			md:       md,
			wantPath: mustParsePath(md, "product.firmware_version"),
			wantFM:   newFieldMask("product.firmware_version"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotFM, err := tt.v.Parse(tt.md)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotPath, tt.wantPath) {
				t.Errorf("Parse() path got = %v, want %v", gotPath, tt.wantPath)
			}
			if diff := cmp.Diff(tt.wantFM, gotFM, protocmp.Transform()); diff != "" {
				t.Errorf("Parse() field mask mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
