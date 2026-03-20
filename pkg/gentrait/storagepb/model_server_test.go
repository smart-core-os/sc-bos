package storagepb

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/storagepb"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

func TestModelServer_GetStorage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	used := uint64(1024)
	model := NewModel()
	_, _ = model.SetStorage(&storagepb.Storage{
		Bytes: &storagepb.StorageBytes{Used: &used},
	})
	server := NewModelServer(model)
	conn := wrap.ServerToClient(storagepb.StorageApi_ServiceDesc, server)
	client := storagepb.NewStorageApiClient(conn)

	got, err := client.GetStorage(ctx, &storagepb.GetStorageRequest{})
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	want := &storagepb.Storage{Bytes: &storagepb.StorageBytes{Used: &used}}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("GetStorage mismatch (-want +got):\n%s", diff)
	}
}

func TestModelServer_PullStorage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	used := uint64(512)
	model := NewModel()
	_, _ = model.SetStorage(&storagepb.Storage{
		Bytes: &storagepb.StorageBytes{Used: &used},
	})
	server := NewModelServer(model)
	conn := wrap.ServerToClient(storagepb.StorageApi_ServiceDesc, server)
	client := storagepb.NewStorageApiClient(conn)

	stream, err := client.PullStorage(ctx, &storagepb.PullStorageRequest{})
	if err != nil {
		t.Fatalf("PullStorage: %v", err)
	}

	// Receive initial value
	res, err := stream.Recv()
	if err != nil {
		t.Fatalf("initial Recv: %v", err)
	}
	if len(res.Changes) == 0 || res.Changes[0].Storage.Bytes.Used == nil || *res.Changes[0].Storage.Bytes.Used != used {
		t.Errorf("expected initial used=%d, got %v", used, res.Changes)
	}

	// Update and receive streaming change
	newUsed := uint64(2048)
	go func() {
		_, _ = model.SetStorage(&storagepb.Storage{
			Bytes: &storagepb.StorageBytes{Used: &newUsed},
		})
	}()
	res, err = stream.Recv()
	if err != nil {
		t.Fatalf("second Recv: %v", err)
	}
	if len(res.Changes) == 0 || res.Changes[0].Storage.Bytes.Used == nil || *res.Changes[0].Storage.Bytes.Used != newUsed {
		t.Errorf("expected updated used=%d, got %v", newUsed, res.Changes)
	}
}

func TestModelServer_PerformStorageAdmin(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	freed := uint64(42)
	var calledWith storagepb.StorageAdminAction
	handler := func(_ context.Context, req *storagepb.PerformStorageAdminRequest) (*storagepb.PerformStorageAdminResponse, error) {
		calledWith = req.Action
		return &storagepb.PerformStorageAdminResponse{FreedItemCount: &freed}, nil
	}

	model := NewModel()
	server := NewModelServer(model, WithAdminHandler(handler))
	conn := wrap.ServerToClient(storagepb.StorageAdminApi_ServiceDesc, server)
	client := storagepb.NewStorageAdminApiClient(conn)

	resp, err := client.PerformStorageAdmin(ctx, &storagepb.PerformStorageAdminRequest{
		Action: storagepb.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR,
	})
	if err != nil {
		t.Fatalf("PerformStorageAdmin: %v", err)
	}
	if calledWith != storagepb.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR {
		t.Errorf("expected CLEAR action, got %v", calledWith)
	}
	if resp.FreedItemCount == nil || *resp.FreedItemCount != freed {
		t.Errorf("expected FreedItemCount=%d, got %v", freed, resp.FreedItemCount)
	}
}

func TestModelServer_PerformStorageAdmin_NoHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	model := NewModel()
	server := NewModelServer(model)
	conn := wrap.ServerToClient(storagepb.StorageAdminApi_ServiceDesc, server)
	client := storagepb.NewStorageAdminApiClient(conn)

	_, err := client.PerformStorageAdmin(ctx, &storagepb.PerformStorageAdminRequest{
		Action: storagepb.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR,
	})
	if status.Code(err) != codes.Unimplemented {
		t.Errorf("expected Unimplemented, got %v", err)
	}
}

func TestModelServer_DescribeStorage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	support := &storagepb.StorageSupport{
		SupportedActions: []storagepb.StorageAdminAction{
			storagepb.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR,
			storagepb.StorageAdminAction_STORAGE_ADMIN_ACTION_DELETE_OLD,
		},
		ItemName: "record",
	}
	model := NewModel()
	server := NewModelServer(model, WithStorageSupport(support))
	conn := wrap.ServerToClient(storagepb.StorageInfo_ServiceDesc, server)
	client := storagepb.NewStorageInfoClient(conn)

	got, err := client.DescribeStorage(ctx, &storagepb.DescribeStorageRequest{})
	if err != nil {
		t.Fatalf("DescribeStorage: %v", err)
	}
	if diff := cmp.Diff(support, got, protocmp.Transform()); diff != "" {
		t.Errorf("DescribeStorage mismatch (-want +got):\n%s", diff)
	}
}

func TestModelServer_HasSubscribers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	model := NewModel()
	server := NewModelServer(model)

	if server.HasSubscribers() {
		t.Fatal("expected no subscribers initially")
	}

	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	conn := wrap.ServerToClient(storagepb.StorageApi_ServiceDesc, server)
	client := storagepb.NewStorageApiClient(conn)

	stream, err := client.PullStorage(streamCtx, &storagepb.PullStorageRequest{})
	if err != nil {
		t.Fatalf("PullStorage: %v", err)
	}
	// Consume the initial value — the server handler is now running and has incremented subscribers.
	if _, err := stream.Recv(); err != nil {
		t.Fatalf("initial Recv: %v", err)
	}
	if !server.HasSubscribers() {
		t.Error("expected HasSubscribers=true while PullStorage stream is active")
	}

	// Cancel the stream and wait for the server handler to exit and decrement the counter.
	streamCancel()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if !server.HasSubscribers() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if server.HasSubscribers() {
		t.Error("expected HasSubscribers=false after stream is closed")
	}
}

func ptr[T any](v T) *T { return &v }
