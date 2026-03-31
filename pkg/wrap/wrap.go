package wrap

import (
	"context"
	"fmt"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var ErrMethodNotFound = status.Error(codes.Unimplemented, "method not found")
var ErrMethodShape = status.Error(codes.Internal, "method stream shape mismatch")

// ServerToClient returns a grpc.ClientConnInterface that can service all methods described by desc.
// srv is the gRPC server implementation type (a type that could be passed to a grpc RegisterXxxServer function).
//
// Unary and streaming calls are supported.
//
// The call options grpc.Header and grpc.Trailer are supported for unary RPCs. All other call options are ignored.
func ServerToClient(desc grpc.ServiceDesc, srv any) grpc.ClientConnInterface {
	// check that srv is the right type to be a server for desc
	expectType := reflect.TypeOf(desc.HandlerType).Elem()
	if !reflect.TypeOf(srv).Implements(expectType) {
		panic(fmt.Sprintf("wrap: srv must be of type %v", expectType))
	}

	methods := make(map[string]grpc.MethodDesc)
	for _, m := range desc.Methods {
		fullName := fmt.Sprintf("/%s/%s", desc.ServiceName, m.MethodName)
		methods[fullName] = m
	}
	streams := make(map[string]grpc.StreamDesc)
	for _, s := range desc.Streams {
		fullName := fmt.Sprintf("/%s/%s", desc.ServiceName, s.StreamName)
		streams[fullName] = s
	}

	return &wrapper{
		methods: methods,
		streams: streams,
		srv:     srv,
	}
}

type wrapper struct {
	methods map[string]grpc.MethodDesc
	streams map[string]grpc.StreamDesc
	srv     any
}

func (w *wrapper) Invoke(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error {
	matched, ok := w.methods[method]
	if !ok {
		return ErrMethodNotFound
	}

	ctx, clientServerStream, ss, cs := w.startStream(ctx, method)
	go func() {
		res, err := matched.Handler(w.srv, ctx, func(dst any) error {
			return ss.RecvMsg(dst)
		}, nil)
		if err != nil {
			clientServerStream.Close(err)
			return
		}
		err = ss.SendMsg(res)
		clientServerStream.Close(err)
	}()

	if err := cs.SendMsg(args); err != nil {
		return err
	}
	if err := cs.CloseSend(); err != nil {
		return err
	}
	err := cs.RecvMsg(reply)

	mdErr := collectMetadata(cs, opts)
	if mdErr != nil && err == nil {
		err = mdErr
	}

	return err
}

func (w *wrapper) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
	matched, ok := w.streams[method]
	if !ok {
		if matchedMethod, ok := w.methods[method]; ok {
			// caller is trying to use a unary method as a stream, this requires special handling
			matched = adaptUnaryToStream(matchedMethod)
		} else {
			return nil, ErrMethodNotFound
		}
	}

	if matched.ServerStreams != desc.ServerStreams || matched.ClientStreams != desc.ClientStreams {
		return nil, ErrMethodShape
	}

	ctx, clientServerStream, ss, cs := w.startStream(ctx, method)
	go func() {
		err := matched.Handler(w.srv, ss)
		clientServerStream.Close(err)
	}()

	return cs, nil
}

func (w *wrapper) startStream(ctx context.Context, method string) (context.Context, *ClientServerStream, grpc.ServerStream, grpc.ClientStream) {
	// convert client's outgoing metadata to server's incoming metadata
	md, _ := metadata.FromOutgoingContext(ctx)
	md = cloneMD(md) // to prevent client from concurrently modifying the metadata

	ctx = metadata.NewIncomingContext(ctx, md)
	// attach a TransportStream to the context, so the server can send headers
	sts := &serverTransportStream{method: method}
	ctx = grpc.NewContextWithServerTransportStream(ctx, sts)

	clientServerStream := NewClientServerStream(ctx)
	ss := clientServerStream.Server()
	sts.ss = ss
	cs := clientServerStream.Client()

	return ctx, clientServerStream, ss, cs
}

// thin implementation of grpc.ServerTransportStream, only to allow grpc.SetHeader to work,
// which relies on a ServerTransportStream attached to the server context
type serverTransportStream struct {
	ss     grpc.ServerStream
	method string
}

func (ts *serverTransportStream) SetHeader(md metadata.MD) error {
	return ts.ss.SetHeader(md)
}

func (ts *serverTransportStream) SendHeader(md metadata.MD) error {
	return ts.ss.SendHeader(md)
}

func (ts *serverTransportStream) SetTrailer(md metadata.MD) error {
	ts.ss.SetTrailer(md)
	return nil
}

func (ts *serverTransportStream) Method() string {
	return ts.method
}

func cloneMD(md metadata.MD) metadata.MD {
	if md == nil {
		return nil
	}
	newMD := make(metadata.MD)
	for k, v := range md {
		newMD[k] = append([]string(nil), v...)
	}
	return newMD
}

// gather headers and trailers into grpc.Header and grpc.Trailer call options
func collectMetadata(cs grpc.ClientStream, opts []grpc.CallOption) error {
	var err error
	for _, opt := range opts {
		switch opt := opt.(type) {
		case grpc.HeaderCallOption:
			hdr, hdrErr := cs.Header()
			if hdrErr != nil && err == nil {
				err = hdrErr
			}
			*opt.HeaderAddr = cloneMD(hdr)

		case grpc.TrailerCallOption:
			trailer := cs.Trailer()
			*opt.TrailerAddr = cloneMD(trailer)
		}
	}
	return err
}

func adaptUnaryToStream(desc grpc.MethodDesc) grpc.StreamDesc {
	return grpc.StreamDesc{
		StreamName:    desc.MethodName,
		ServerStreams: false,
		ClientStreams: false,
		Handler: func(srv any, stream grpc.ServerStream) error {
			dec := func(dst any) error {
				return stream.RecvMsg(dst)
			}
			res, err := desc.Handler(srv, stream.Context(), dec, nil)
			if err != nil {
				return err
			}
			return stream.SendMsg(res)
		},
	}
}
