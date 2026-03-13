package wrap

import (
	"context"
	"errors"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/internal/testproto"
)

func Test_clientSend(t *testing.T) {
	ctx := context.Background()
	clientServer := NewClientServerStream(ctx)
	client := clientServer.Client()
	server := clientServer.Server()

	sentMessage := &traits.Brightness{
		LevelPercent: 12,
	}
	receivedMessage := &traits.Brightness{}

	go client.SendMsg(sentMessage)
	server.RecvMsg(receivedMessage)

	if !reflect.DeepEqual(sentMessage, receivedMessage) {
		t.Errorf("%v != %v", sentMessage, receivedMessage)
	}
}

func Test_serverSend(t *testing.T) {
	ctx := context.Background()
	clientServer := NewClientServerStream(ctx)
	client := clientServer.Client()
	server := clientServer.Server()

	sentMessage := &traits.Brightness{
		LevelPercent: 12,
	}
	receivedMessage := &traits.Brightness{}

	go server.SendMsg(sentMessage)
	client.RecvMsg(receivedMessage)

	if !reflect.DeepEqual(sentMessage, receivedMessage) {
		t.Errorf("%v != %v", sentMessage, receivedMessage)
	}
}

func TestClientServerStream_headerReturnsOnClose(t *testing.T) {
	t.Run("nil err", func(t *testing.T) {
		assertHeaderErrOnClose(t, nil, nil)
	})
	t.Run("non-nil err", func(t *testing.T) {
		err := errors.New("early closed")
		assertHeaderErrOnClose(t, err, nil)
	})
}

// tests that the server can send headers after receiving a message from the client
// This is a regression test to detect a bug in ClientServerStream where calling ServerStream.RecvMsg (including the
// implicit RecvMsg for the request in a server-streaming RPC) would send the headers to the client immediately,
// which is not the expected behavior.
func TestClientServerStream_HeadersAfterRecv(t *testing.T) {
	s := NewClientServerStream(context.Background())
	client := s.Client()
	server := s.Server()

	var grp sync.WaitGroup
	grp.Add(2)

	// client code
	// simulate a client performing a server-streaming call:
	// send the request message immediately, then close the stream
	go func() {
		defer grp.Done()
		err := client.SendMsg(&emptypb.Empty{})
		if err != nil {
			t.Errorf("client SendMsg: %v", err)
		}
		err = client.CloseSend()
		if err != nil {
			t.Errorf("client CloseSend: %v", err)
		}

		// receive first message, so that headers are available
		err = client.RecvMsg(&emptypb.Empty{})
		if err != nil {
			t.Errorf("client RecvMsg: %v", err)
		}
		md, err := client.Header()
		if err != nil {
			t.Errorf("client Header error: %v", err)
		}
		expectMD := metadata.Pairs("a", "avalue", "b", "bvalue")
		if !maps.EqualFunc(md, expectMD, slices.Equal[[]string]) {
			t.Errorf("client Header = %v; want %v", md, expectMD)
		}
	}()

	// server code
	// simulate a server performing a server-streaming RPC:
	// first receive the request message, then set some headers, then send a response
	go func() {
		defer grp.Done()

		req := &emptypb.Empty{}
		err := server.RecvMsg(req)
		if err != nil {
			t.Errorf("server RecvMsg: %v", err)
		}

		err = server.SendHeader(metadata.Pairs("a", "avalue", "b", "bvalue"))
		if err != nil {
			t.Errorf("server SetHeader: %v", err)
		}

		err = server.SendMsg(&emptypb.Empty{})
		if err != nil {
			t.Errorf("server SendMsg: %v", err)
		}
	}()

	grp.Wait()
}

// regression test: because of nondeterministic select-cases when multiple cases are ready,
// ClientStream.Header() sometimes returned nil after the stream was closed, even if headers had been received.
func TestClientServerStream_HeadersAfterClose(t *testing.T) {
	// doesn't occur every time, use brute force to improve our chances of catching it
	for i := 0; i < 20 && !t.Failed(); i++ {
		s := NewClientServerStream(context.Background())
		cs := s.Client()
		ss := s.Server()

		err := ss.SendHeader(metadata.Pairs("hello", "world"))
		if err != nil {
			t.Fatalf("SendHeader: %v", err)
		}
		s.Close(errors.New("test close"))

		md, err := cs.Header()
		if err != nil {
			t.Errorf("unexpected ClientStream.Header error %v", err)
		}
		expectMD := metadata.Pairs("hello", "world")
		if diff := cmp.Diff(expectMD, md); diff != "" {
			t.Errorf("ClientStream.Header mismatch (-want +got):\n%s", diff)
		}
	}
}

func assertHeaderErrOnClose(t *testing.T, closeErr, wantErr error) {
	ctx := context.Background()
	cs := NewClientServerStream(ctx)
	client := cs.Client()

	go cs.Close(closeErr)

	var gotErr error
	headerDone := make(chan struct{})
	go func() {
		_, gotErr = client.Header()
		close(headerDone)
	}()

	select {
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for Client.Header()")
	case <-headerDone:
	}
	if gotErr != wantErr {
		t.Fatalf("Header err want %v, got %v", wantErr, gotErr)
	}
}

func TestClientServerStream_RecvMsg_errOnClose(t *testing.T) {
	t.Run("nil err", func(t *testing.T) {
		assertRecvMsgErrOnClose(t, nil, io.EOF)
	})
	t.Run("non-nil err", func(t *testing.T) {
		err := errors.New("early closed")
		assertRecvMsgErrOnClose(t, err, err)
	})
}

func TestClientServerStream_ModifyMessageOnClientSend(t *testing.T) {
	cs := NewClientServerStream(t.Context())
	defer cs.Close(nil)
	client := cs.Client()
	server := cs.Server()

	wg := sync.WaitGroup{}
	wg.Add(1)
	originalMsg := &testproto.TestAllTypes{MapStringBytes: map[string][]byte{"key1": []byte("value1")}}

	// client modifies the message after sending
	go func() {
		defer wg.Done()
		err := server.RecvMsg(&testproto.TestAllTypes{})
		if err != nil {
			t.Errorf("Server RecvMsg: %v", err)
		}
	}()

	err := client.SendMsg(originalMsg)
	if err != nil {
		t.Errorf("Client SendMsg: %v", err)
	}

	// Modify the original message after sending.
	// ClientServerStream will cause race conditions attempting to read the map whilst this modification is happening
	// unless the proto.Message is cloned on SendMsg.
	originalMsg.MapStringBytes["key1"] = []byte("value2")

	wg.Wait()
}

func TestClientServerStream_ModifyMessageOnServerSend(t *testing.T) {
	cs := NewClientServerStream(t.Context())
	defer cs.Close(nil)
	client := cs.Client()
	server := cs.Server()

	wg := sync.WaitGroup{}
	wg.Add(1)

	originalMsg := &testproto.TestAllTypes{MapStringBytes: map[string][]byte{"key1": []byte("value1")}}

	// server modifies the message after sending
	go func() {
		defer wg.Done()
		err := client.RecvMsg(&testproto.TestAllTypes{})
		if err != nil {
			t.Errorf("Client RecvMsg: %v", err)
		}
	}()

	err := server.SendMsg(originalMsg)
	if err != nil {
		t.Errorf("Server SendMsg: %v", err)
	}
	// Modify the original message after sending.
	// ClientServerStream will cause race conditions attempting to read the map whilst this modification is happening
	// unless the proto.Message is cloned on SendMsg.
	originalMsg.MapStringBytes["key1"] = []byte("value2")
	wg.Wait()
}

func assertRecvMsgErrOnClose(t *testing.T, closeErr, wantErr error) {
	ctx := context.Background()
	cs := NewClientServerStream(ctx)
	client := cs.Client()

	go cs.Close(closeErr)

	var gotErr error
	gotMsg := &emptypb.Empty{}
	recvDone := make(chan struct{})
	go func() {
		gotErr = client.RecvMsg(gotMsg)
		close(recvDone)
	}()

	select {
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for Client.RecvMsg()")
	case <-recvDone:
	}
	if gotErr != wantErr {
		t.Fatalf("RecvMsg err want %v, got %v", wantErr, gotErr)
	}
}
