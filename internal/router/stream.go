package router

import (
	"context"
	"errors"
	"io"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type MethodResolver interface {
	ResolveMethod(fullName string) (Method, error)
}

// StreamHandler returns a grpc.StreamHandler that routes incoming message to the appropriate downstream target.
func StreamHandler(r MethodResolver) grpc.StreamHandler {
	return func(_ any, serverStream grpc.ServerStream) error {
		method, ok := grpc.Method(serverStream.Context())
		if !ok {
			return ErrMissingMethod
		}
		target, err := r.ResolveMethod(method)
		if err != nil {
			return err
		}

		md, _ := metadata.FromIncomingContext(serverStream.Context())
		// The Connection header (i.e. Connection: keep-alive), if present in the request we send to the downstream server,
		// will cause the gRPC internals to abort the connection returning an error like:
		// > stream terminated by RST_STREAM with error code: PROTOCOL_ERROR
		// This header is added automatically by browsers via grpc-web, but the grpc-web wrapper isn't removing it.
		//
		// See https://github.com/grpc/grpc-go/blob/c04b085930ce33ee83cc3f92dbe7632031e127a9/internal/transport/http2_server.go#L444
		delete(md, "connection")
		// The end user's bearer token has already been checked by this node's policy interceptor.
		// The downstream hop is authenticated by this node's client certificate, and the remote node usually can't
		// verify a token issued for us, so forwarding it only leaks it and produces verification noise.
		//
		// See SCB-1514.
		delete(md, "authorization")
		ctx := metadata.NewOutgoingContext(serverStream.Context(), md)

		clientCtx, stopClient := context.WithCancel(ctx)
		defer stopClient()

		// set up (potentially) delayed client stream resolution
		clientStream := &clientStream{notify: make(chan struct{})}
		resolver := func(mr MsgRecver) (grpc.ClientStream, error) {
			cc, err := target.Resolver.ResolveConn(mr)
			if err != nil {
				return nil, err
			}
			return cc.NewStream(clientCtx, descriptorToStreamDesc(target.Desc), method)
		}

		c2sErr, s2cErr := make(chan error, 1), make(chan error, 1) // buffered so we don't leak the following go routines
		go func() { c2sErr <- streamClientToServer(serverStream, clientStream, target.Desc.Output()) }()
		go func() { s2cErr <- streamServerToClient(clientStream, serverStream, resolver, target.Desc.Input()) }()

		for range 2 {
			select {
			case err := <-c2sErr:
				// trailers are only available from the client once it's done (has returned an error)
				serverStream.SetTrailer(clientStream.Trailer(clientCtx))
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err // will be the error returned from the client stream
			case err := <-s2cErr:
				if !errors.Is(err, io.EOF) {
					// unexpected error (network or something), so stop the client stream and raise an error
					stopClient()
					return status.Convert(err).Err()
				}

				// In the expected path, we got a signal so say the server won't be sending anything else.
				// We let the client know, even though the client -> server side of the stream may continue to send.
				_ = clientStream.CloseSend(clientCtx)
			}
		}

		return status.Errorf(codes.Internal, "unexpected path, bad stream state transition")
	}
}

// streamClientToServer forwards messages (and headers) from src to dst.
func streamClientToServer(dst grpc.ServerStream, src *clientStream, msgDesc protoreflect.MessageDescriptor) error {
	client, err := src.Get(dst.Context())
	if err != nil {
		return err
	}
	for first := true; ; first = false {
		// We need a new msg for each loop because it is unsafe to modify m after passing to dst.SendMsg.
		// See the docs for [grpc.ServerStream.SendMsg].
		m := dynamicpb.NewMessage(msgDesc)
		if err := client.RecvMsg(m); err != nil {
			return err
		}
		if first {
			// Handle headers before we forward the first message.
			// We do it here because the headers only exist on src after we've received the first message.
			md, err := client.Header()
			if err != nil {
				return err
			}
			if err := dst.SendHeader(md); err != nil {
				return err
			}
		}
		if err := dst.SendMsg(m); err != nil {
			return err
		}
	}
}

// streamResolver is like a ConnResolver but resolves to a grpc.ClientStream.
type streamResolver func(MsgRecver) (grpc.ClientStream, error)

// streamServerToClient forwards messages from src to dst.
// The first message we receive from src is used to determine which client to forward to via resolver.
func streamServerToClient(dst *clientStream, src grpc.ServerStream, resolver streamResolver, msgDesc protoreflect.MessageDescriptor) error {
	// these are filled via resolver after we've received the first message from src.
	var (
		m      proto.Message
		client grpc.ClientStream
	)
	for {
		if client == nil { // could also be m == nil
			// first message, let's try and figure out who to send it to
			var err error
			msgCap := &captureMsgRecver{MsgRecver: src}
			client, err = resolver(msgCap)
			if err != nil {
				return err
			}
			if msg, ok := msgCap.msg.(proto.Message); ok {
				// this case means the resolver read a message to do the resolve
				m = msg
			} else if msgCap.msg == nil {
				// this case means no message was read, so we need to do it ourselves
				m = dynamicpb.NewMessage(msgDesc)
				if err := src.RecvMsg(m); err != nil {
					return err
				}
			} else {
				// the resolver read a message but didn't pass a proto.Message, we don't support this
				return status.Errorf(codes.Internal, "a message that is not a proto.Message passed to RecvMsg")
			}
			dst.Set(client)
		} else {
			// We need a new msg for each loop because it is unsafe to modify m after passing to client.SendMsg.
			// See the docs for [grpc.ClientStream.SendMsg].
			m = m.ProtoReflect().New().Interface()
			if err := src.RecvMsg(m); err != nil {
				return err
			}
		}
		if err := client.SendMsg(m); err != nil {
			return err
		}
	}
}

// clientStream is a wrapper around grpc.ClientStream that allows for delayed stream creation.
type clientStream struct {
	mu     sync.Mutex
	cc     grpc.ClientStream
	notify chan struct{}
}

func (cc *clientStream) Set(cs grpc.ClientStream) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	if cc.cc != nil {
		panic("client stream already set")
	}
	cc.cc = cs
	close(cc.notify)
}

func (cc *clientStream) Get(ctx context.Context) (grpc.ClientStream, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-cc.notify:
		// no need to lock, guaranteed to be set once notify is closed
		return cc.cc, nil
	}
}

func (cc *clientStream) Trailer(ctx context.Context) metadata.MD {
	cs, err := cc.Get(ctx)
	if err != nil {
		return nil
	}
	return cs.Trailer()
}

func (cc *clientStream) CloseSend(ctx context.Context) error {
	cs, err := cc.Get(ctx)
	if err != nil {
		return err
	}
	return cs.CloseSend()
}

func descriptorToStreamDesc(desc protoreflect.MethodDescriptor) *grpc.StreamDesc {
	return &grpc.StreamDesc{
		StreamName:    string(desc.Name()),
		ClientStreams: desc.IsStreamingClient(),
		ServerStreams: desc.IsStreamingServer(),
	}
}
