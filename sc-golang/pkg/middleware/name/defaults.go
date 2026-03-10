package name

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// IfAbsentUnaryInterceptor defines a server unary interceptor that sets the request name to name if it is unset.
// Useful to implement the Smart Core property that an absent name should refer to the server serving the request.
func IfAbsentUnaryInterceptor(name string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		replaceEmptyNameField(req, name)
		return handler(ctx, req)
	}
}

// IfAbsentStreamInterceptor defines a server stream interceptor that sets the request name to name if it is unset.
// Useful to implement the Smart Core property that an absent name should refer to the server serving the request.
func IfAbsentStreamInterceptor(name string) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &absentNameReplaceServerStream{
			ServerStream: ss,
			name:         name,
		})
	}
}

func replaceEmptyNameField(req any, name string) {
	msg, ok := req.(proto.Message)
	if !ok {
		return // not a proto.Message
	}
	fields := msg.ProtoReflect().Descriptor().Fields()
	nameField := fields.ByTextName("name")
	if nameField == nil {
		return // no name field
	}
	if nameField.Kind() != protoreflect.StringKind {
		return // not a string
	}
	nameValue := msg.ProtoReflect().Get(nameField)
	if nameValue.String() != "" {
		return // name isn't empty/default
	}
	msg.ProtoReflect().Set(nameField, protoreflect.ValueOfString(name))
}

type absentNameReplaceServerStream struct {
	grpc.ServerStream
	name string
}

func (w *absentNameReplaceServerStream) RecvMsg(m any) error {
	err := w.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}
	replaceEmptyNameField(m, w.name)
	return nil
}
