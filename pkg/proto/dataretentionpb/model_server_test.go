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

// testBackend is a Backend implementation for tests.
type testBackend struct {
	purge func(ctx context.Context, before *time.Time) (uint64, error)
}

func (b *testBackend) Purge(ctx context.Context, before *time.Time) (uint64, error) {
	return b.purge(ctx, before)
}

// compactBackend extends testBackend with Compact support.
type compactBackend struct {
	testBackend
	compact func(ctx context.Context) error
}

func (b *compactBackend) Compact(ctx context.Context) error {
	return b.compact(ctx)
}

func TestModelServer_GetDataRetention(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	used := uint64(1024)
	model := NewModel()
	_, _ = model.SetDataRetention(&DataRetention{
		Bytes: &DataRetentionBytes{Used: &used},
	})
	server := NewModelServer(model, nil)
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
	server := NewModelServer(model, nil)
	conn := wrap.ServerToClient(DataRetentionApi_ServiceDesc, server)
	client := NewDataRetentionApiClient(conn)

	stream, err := client.PullDataRetention(ctx, &PullDataRetentionRequest{})
	if err != nil {
		t.Fatalf("PullDataRetention: %v", err)
	}

	res, err := stream.Recv()
	if err != nil {
		t.Fatalf("initial Recv: %v", err)
	}
	if len(res.Changes) == 0 || res.Changes[0].DataRetention.Bytes.Used == nil || *res.Changes[0].DataRetention.Bytes.Used != used {
		t.Errorf("expected initial used=%d, got %v", used, res.Changes)
	}

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
	var gotBefore *time.Time
	backend := &testBackend{
		purge: func(_ context.Context, before *time.Time) (uint64, error) {
			gotBefore = before
			return freed, nil
		},
	}

	server := NewModelServer(NewModel(), backend)
	conn := wrap.ServerToClient(DataRetentionApi_ServiceDesc, server)
	client := NewDataRetentionApiClient(conn)

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
	wantBefore := time.Now().Add(-7 * 24 * time.Hour).UTC().Truncate(time.Microsecond)
	var gotBefore *time.Time
	backend := &testBackend{
		purge: func(_ context.Context, before *time.Time) (uint64, error) {
			gotBefore = before
			return freed, nil
		},
	}

	server := NewModelServer(NewModel(), backend)
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
	if !gotBefore.Equal(wantBefore) {
		t.Errorf("expected before=%v, got %v", wantBefore, *gotBefore)
	}
	if resp.FreedItemCount == nil || *resp.FreedItemCount != freed {
		t.Errorf("expected FreedItemCount=%d, got %v", freed, resp.FreedItemCount)
	}
}

func TestModelServer_PurgeDataRetention_NoBackend(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server := NewModelServer(NewModel(), nil)
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

	compacted := false
	backend := &compactBackend{
		testBackend: testBackend{purge: func(_ context.Context, _ *time.Time) (uint64, error) { return 0, nil }},
		compact:     func(_ context.Context) error { compacted = true; return nil },
	}

	server := NewModelServer(NewModel(), backend)
	conn := wrap.ServerToClient(DataRetentionApi_ServiceDesc, server)
	client := NewDataRetentionApiClient(conn)

	_, err := client.CompactDataRetention(ctx, &CompactDataRetentionRequest{})
	if err != nil {
		t.Fatalf("CompactDataRetention: %v", err)
	}
	if !compacted {
		t.Error("expected Compact to be called")
	}
}

func TestModelServer_CompactDataRetention_NoCompact(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// testBackend does not implement Compacter
	backend := &testBackend{purge: func(_ context.Context, _ *time.Time) (uint64, error) { return 0, nil }}
	server := NewModelServer(NewModel(), backend)
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

	t.Run("nil backend", func(t *testing.T) {
		server := NewModelServer(NewModel(), nil)
		conn := wrap.ServerToClient(DataRetentionInfo_ServiceDesc, server)
		got, err := NewDataRetentionInfoClient(conn).DescribeDataRetention(ctx, &DescribeDataRetentionRequest{})
		if err != nil {
			t.Fatalf("DescribeDataRetention: %v", err)
		}
		want := &DataRetentionSupport{}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("purge only", func(t *testing.T) {
		backend := &testBackend{purge: func(_ context.Context, _ *time.Time) (uint64, error) { return 0, nil }}
		server := NewModelServer(NewModel(), backend, WithItemName("record"))
		conn := wrap.ServerToClient(DataRetentionInfo_ServiceDesc, server)
		got, err := NewDataRetentionInfoClient(conn).DescribeDataRetention(ctx, &DescribeDataRetentionRequest{})
		if err != nil {
			t.Fatalf("DescribeDataRetention: %v", err)
		}
		want := &DataRetentionSupport{CanPurge: true, ItemName: "record"}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("purge and compact", func(t *testing.T) {
		backend := &compactBackend{
			testBackend: testBackend{purge: func(_ context.Context, _ *time.Time) (uint64, error) { return 0, nil }},
			compact:     func(_ context.Context) error { return nil },
		}
		server := NewModelServer(NewModel(), backend, WithItemName("row"))
		conn := wrap.ServerToClient(DataRetentionInfo_ServiceDesc, server)
		got, err := NewDataRetentionInfoClient(conn).DescribeDataRetention(ctx, &DescribeDataRetentionRequest{})
		if err != nil {
			t.Fatalf("DescribeDataRetention: %v", err)
		}
		want := &DataRetentionSupport{CanPurge: true, CanCompact: true, ItemName: "row"}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}
