package proxy

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/gen"
)

func TestChildrenFetcher_Poll(t *testing.T) {
	tests := []struct {
		name      string
		initial   map[string]*gen.Device
		responses []*gen.ListDevicesResponse
		wantAdds  []string
		wantUpdts []string
		wantRems  []string
		wantErr   bool
	}{
		{
			name:    "empty response",
			initial: nil,
			responses: []*gen.ListDevicesResponse{
				{Devices: []*gen.Device{}},
			},
			wantAdds: nil,
		},
		{
			name:    "new devices added",
			initial: nil,
			responses: []*gen.ListDevicesResponse{
				{
					Devices: []*gen.Device{
						{Name: "device1", Metadata: &traits.Metadata{}},
						{Name: "device2", Metadata: &traits.Metadata{}},
					},
				},
			},
			wantAdds: []string{"device1", "device2"},
		},
		{
			name: "devices updated",
			initial: map[string]*gen.Device{
				"device1": {Name: "device1", Metadata: &traits.Metadata{}},
			},
			responses: []*gen.ListDevicesResponse{
				{
					Devices: []*gen.Device{
						{Name: "device1", Metadata: &traits.Metadata{
							Traits: []*traits.TraitMetadata{{Name: "OnOff"}},
						}},
					},
				},
			},
			wantUpdts: []string{"device1"},
		},
		{
			name: "devices removed",
			initial: map[string]*gen.Device{
				"device1": {Name: "device1", Metadata: &traits.Metadata{}},
				"device2": {Name: "device2", Metadata: &traits.Metadata{}},
			},
			responses: []*gen.ListDevicesResponse{
				{
					Devices: []*gen.Device{
						{Name: "device1", Metadata: &traits.Metadata{}},
					},
				},
			},
			wantRems: []string{"device2"},
		},
		{
			name: "no change if proto equal",
			initial: map[string]*gen.Device{
				"device1": {Name: "device1", Metadata: &traits.Metadata{}},
			},
			responses: []*gen.ListDevicesResponse{
				{
					Devices: []*gen.Device{
						{Name: "device1", Metadata: &traits.Metadata{}},
					},
				},
			},
		},
		{
			name:    "paginated response",
			initial: nil,
			responses: []*gen.ListDevicesResponse{
				{
					Devices: []*gen.Device{
						{Name: "device1", Metadata: &traits.Metadata{}},
					},
					NextPageToken: "page2",
				},
				{
					Devices: []*gen.Device{
						{Name: "device2", Metadata: &traits.Metadata{}},
					},
				},
			},
			wantAdds: []string{"device1", "device2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockDevicesClient{responses: tt.responses}
			fetcher := &childrenFetcher{
				client: client,
				name:   "test-parent",
				known:  tt.initial,
			}

			ctx := context.Background()
			changes := make(chan *gen.PullDevicesResponse_Change, 10)

			err := fetcher.Poll(ctx, changes)
			close(changes)

			if (err != nil) != tt.wantErr {
				t.Errorf("Poll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var gotAdds []string
			var gotUpdts []string
			var gotRems []string

			for change := range changes {
				switch change.Type {
				case types.ChangeType_ADD:
					gotAdds = append(gotAdds, change.NewValue.Name)
				case types.ChangeType_UPDATE:
					gotUpdts = append(gotUpdts, change.NewValue.Name)
				case types.ChangeType_REMOVE:
					gotRems = append(gotRems, change.OldValue.Name)
				}
			}

			if !stringSlicesEqual(gotAdds, tt.wantAdds) {
				t.Errorf("Poll() adds = %v, want %v", gotAdds, tt.wantAdds)
			}
			if !stringSlicesEqual(gotUpdts, tt.wantUpdts) {
				t.Errorf("Poll() updates = %v, want %v", gotUpdts, tt.wantUpdts)
			}
			if !stringSlicesEqual(gotRems, tt.wantRems) {
				t.Errorf("Poll() removes = %v, want %v", gotRems, tt.wantRems)
			}
		})
	}
}

func TestChildrenFetcher_Poll_Error(t *testing.T) {
	client := &mockDevicesClient{err: errors.New("network error")}
	fetcher := &childrenFetcher{
		client: client,
		name:   "test-parent",
	}

	ctx := context.Background()
	changes := make(chan *gen.PullDevicesResponse_Change, 10)

	err := fetcher.Poll(ctx, changes)
	if err == nil {
		t.Error("Poll() expected error, got nil")
	}
}

func TestChildrenFetcher_Pull(t *testing.T) {
	t.Run("streams changes", func(t *testing.T) {
		stream := &mockPullDevicesClient{
			changes: []*gen.PullDevicesResponse{
				{
					Changes: []*gen.PullDevicesResponse_Change{
						{
							Type:     types.ChangeType_ADD,
							NewValue: &gen.Device{Name: "device1"},
						},
					},
				},
				{
					Changes: []*gen.PullDevicesResponse_Change{
						{
							Type:     types.ChangeType_UPDATE,
							NewValue: &gen.Device{Name: "device1"},
						},
					},
				},
			},
		}

		client := &mockDevicesClient{stream: stream}
		fetcher := &childrenFetcher{
			client: client,
			name:   "test-parent",
		}

		ctx, cancel := context.WithCancel(context.Background())
		changes := make(chan *gen.PullDevicesResponse_Change, 10)

		errCh := make(chan error, 1)
		go func() {
			errCh <- fetcher.Pull(ctx, changes)
		}()

		// Read the changes
		var got []*gen.PullDevicesResponse_Change
		for i := 0; i < 2; i++ {
			select {
			case change := <-changes:
				got = append(got, change)
			}
		}

		cancel()
		<-errCh

		if len(got) != 2 {
			t.Errorf("Pull() got %d changes, want 2", len(got))
		}
	})

	t.Run("handles stream error", func(t *testing.T) {
		stream := &mockPullDevicesClient{
			err: errors.New("stream error"),
		}

		client := &mockDevicesClient{stream: stream}
		fetcher := &childrenFetcher{
			client: client,
			name:   "test-parent",
		}

		ctx := context.Background()
		changes := make(chan *gen.PullDevicesResponse_Change, 10)

		err := fetcher.Pull(ctx, changes)
		if err == nil {
			t.Error("Pull() expected error, got nil")
		}
	})
}

type mockDevicesClient struct {
	gen.DevicesApiClient
	responses []*gen.ListDevicesResponse
	stream    *mockPullDevicesClient
	err       error
	callCount int
}

func (m *mockDevicesClient) ListDevices(_ context.Context, _ *gen.ListDevicesRequest, _ ...grpc.CallOption) (*gen.ListDevicesResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.callCount >= len(m.responses) {
		return &gen.ListDevicesResponse{}, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return resp, nil
}

func (m *mockDevicesClient) PullDevices(_ context.Context, _ *gen.PullDevicesRequest, _ ...grpc.CallOption) (gen.DevicesApi_PullDevicesClient, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.stream, nil
}

type mockPullDevicesClient struct {
	gen.DevicesApi_PullDevicesClient
	changes []*gen.PullDevicesResponse
	index   int
	err     error
}

func (m *mockPullDevicesClient) Recv() (*gen.PullDevicesResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.index >= len(m.changes) {
		return nil, context.Canceled
	}
	msg := m.changes[m.index]
	m.index++
	return msg, nil
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	seen := make(map[string]int)
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}
