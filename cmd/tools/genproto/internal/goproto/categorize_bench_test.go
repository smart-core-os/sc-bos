package goproto

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func BenchmarkDetermineGeneratorsFromDescriptor(b *testing.B) {
	desc := &descriptorpb.FileDescriptorProto{
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
	}

	b.ResetTimer()
	for b.Loop() {
		_ = determineGeneratorsFromDescriptor(desc)
	}
}
