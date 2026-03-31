package goproto

import (
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"
)

func BenchmarkDetermineGeneratorsFromDescriptor(b *testing.B) {
	desc := &descriptorpb.FileDescriptorProto{
		Name:    new("test.proto"),
		Package: new("test"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: new("GetRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   new("name"),
						Number: new(int32(1)),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
			{
				Name: new("UpdateRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   new("name"),
						Number: new(int32(1)),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: new("TestService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:      new("Get"),
						InputType: new(".test.GetRequest"),
					},
					{
						Name:      new("Update"),
						InputType: new(".test.UpdateRequest"),
					},
				},
			},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_ = determineGeneratorsFromDescriptor(desc)
	}
}
