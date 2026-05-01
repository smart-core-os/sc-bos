package dataretentionpb

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

func TestModelServer_GetDataRetention(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	used := uint64(1024)
	model := NewModel()
	_, _ = model.SetDataRetention(&DataRetention{
		Bytes: &DataRetentionBytes{Used: &used},
	})
	server := NewModelServer(model)
	conn := wrap.ServerToClient(DataRetentionApi_ServiceDesc, server)
	client := NewDataRetentionApiClient(conn)

	got, err := client.GetDataRetention(ctx, &GetDataRetentionRequest{})
	if err != nil {
		t.Fatalf("GetDataRetention: %v", err)
	}
	want := &DataRetention{Bytes: &DataRetentionBytes{Used: &used}}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("GetDataRetention mismatch (-want +got):\n%s", diff)
	}
}

func TestModelServer_PullDataRetention(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	used := uint64(512)
	model := NewModel()
	_, _ = model.SetDataRetention(&DataRetention{
		Bytes: &DataRetentionBytes{Used: &used},
	})
	server := NewModelServer(model)
	conn := wrap.ServerToClient(DataRetentionApi_ServiceDesc, server)
	client := NewDataRetentionApiClient(conn)

	stream, err := client.PullDataRetention(ctx, &PullDataRetentionRequest{})
	if err != nil {
		t.Fatalf("PullDataRetention: %v", err)
	}

	// Receive initial value
	res, err := stream.Recv()
	if err != nil {
		t.Fatalf("initial Recv: %v", err)
	}
	if len(res.Changes) == 0 || res.Changes[0].DataRetention.Bytes.Used == nil || *res.Changes[0].DataRetention.Bytes.Used != used {
		t.Errorf("expected initial used=%d, got %v", used, res.Changes)
	}

	// Update and receive streaming change
	newUsed := uint64(2048)
	go func() {
		_, _ = model.SetDataRetention(&DataRetention{
			Bytes: &DataRetentionBytes{Used: &newUsed},
		})
	}()
	res, err = stream.Recv()
	if err != nil {
		t.Fatalf("second Recv: %v", err)
	}
	if len(res.Changes) == 0 || res.Changes[0].DataRetention.Bytes.Used == nil || *res.Changes[0].DataRetention.Bytes.Used != newUsed {
		t.Errorf("expected updated used=%d, got %v", newUsed, res.Changes)
	}
}

func TestModelServer_PurgeDataRetention(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	freed := uint64(42)
	var gotBefore *timestamppb.Timestamp
	handler := func(_ context.Context, req *PurgeDataRetentionRequest) (*PurgeDataRetentionResponse, error) {
		gotBefore = req.Before
		return &PurgeDataRetentionResponse{FreedItemCount: &freed}, nil
	}

	model := NewModel()
	server := NewModelServer(model, WithPurgeHandler(handler))
	conn := wrap.ServerToClient(DataRetentionApi_ServiceDesc, server)
	client := NewDataRetentionApiClient(conn)

	// Purge all (no before)
	resp, err := client.PurgeDataRetention(ctx, &PurgeDataRetentionRequest{})
	if err != nil {
		t.Fatalf("PurgeDataRetention: %v", err)
	}
	if gotBefore != nil {
		t.Errorf("expected before=nil for full purge, got %v", gotBefore)
	}
	if resp.FreedItemCount == nil || *resp.FreedItemCount != freed {
		t.Errorf("expected FreedItemCount=%d, got %v", freed, resp.FreedItemCount)
	}
}

func TestModelServer_PurgeDataRetention_WithBefore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	freed := uint64(10)
	wantBefore := time.Now().Add(-7 * 24 * time.Hour)
	var gotBefore *timestamppb.Timestamp
	handler := func(_ context.Context, req *PurgeDataRetentionRequest) (*PurgeDataRetentionResponse, error) {
		gotBefore = req.Before
		return &PurgeDataRetentionResponse{FreedItemCount: &freed}, nil
	}

	model := NewModel()
	server := NewModelServer(model, WithPurgeHandler(handler))
	conn := wrap.ServerToClient(DataRetentionApi_ServiceDesc, server)
	client := NewDataRetentionApiClient(conn)

	resp, err := client.PurgeDataRetention(ctx, &PurgeDataRetentionRequest{
		Before: timestamppb.New(wantBefore),
	})
	if err != nil {
		t.Fatalf("PurgeDataRetention: %v", err)
	}
	if gotBefore == nil {
		t.Fatal("expected before to be set")
	}
	if !gotBefore.AsTime().Equal(wantBefore.UTC()) {
		t.Errorf("expected before=%v, got %v", wantBefore, gotBefore.AsTime())
	}
	if resp.FreedItemCount == nil || *resp.FreedItemCount != freed {
		t.Errorf("expected FreedItemCount=%d, got %v", freed, resp.FreedItemCount)
	}
}

func TestModelServer_PurgeDataRetention_NoHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	model := NewModel()
	server := NewModelServer(model)
	conn := wrap.ServerToClient(DataRetentionApi_ServiceDesc, server)
	client := NewDataRetentionApiClient(conn)

	_, err := client.PurgeDataRetention(ctx, &PurgeDataRetentionRequest{})
	if status.Code(err) != codes.Unimplemented {
		t.Errorf("expected Unimplemented, got %v", err)
	}
}

func TestModelServer_CompactDataRetention(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	freed := uint64(1024)
	handler := func(_ context.Context, _ *CompactDataRetentionRequest) (*CompactDataRetentionResponse, error) {
		return &CompactDataRetentionResponse{FreedByteCount: &freed}, nil
	}

	model := NewModel()
	server := NewModelServer(model, WithCompactHandler(handler))
	conn := wrap.ServerToClient(DataRetentionApi_ServiceDesc, server)
	client := NewDataRetentionApiClient(conn)

	resp, err := client.CompactDataRetention(ctx, &CompactDataRetentionRequest{})
	if err != nil {
		t.Fatalf("CompactDataRetention: %v", err)
	}
	if resp.FreedByteCount == nil || *resp.FreedByteCount != freed {
		t.Errorf("expected FreedByteCount=%d, got %v", freed, resp.FreedByteCount)
	}
}

func TestModelServer_CompactDataRetention_NoHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	model := NewModel()
	server := NewModelServer(model)
	conn := wrap.ServerToClient(DataRetentionApi_ServiceDesc, server)
	client := NewDataRetentionApiClient(conn)

	_, err := client.CompactDataRetention(ctx, &CompactDataRetentionRequest{})
	if status.Code(err) != codes.Unimplemented {
		t.Errorf("expected Unimplemented, got %v", err)
	}
}

func TestModelServer_DescribeDataRetention(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	support := &DataRetentionSupport{
		CanPurge: true,
		ItemName: "record",
	}
	model := NewModel()
	server := NewModelServer(model, WithDataRetentionSupport(support))
	conn := wrap.ServerToClient(DataRetentionInfo_ServiceDesc, server)
	client := NewDataRetentionInfoClient(conn)

	got, err := client.DescribeDataRetention(ctx, &DescribeDataRetentionRequest{})
	if err != nil {
		t.Fatalf("DescribeDataRetention: %v", err)
	}
	if diff := cmp.Diff(support, got, protocmp.Transform()); diff != "" {
		t.Errorf("DescribeDataRetention mismatch (-want +got):\n%s", diff)
	}
}
