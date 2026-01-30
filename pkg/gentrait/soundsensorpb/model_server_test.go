package soundsensorpb

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

func TestModelServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	model := NewModel(resource.WithInitialValue(&soundsensorpb.SoundLevel{
		SoundPressureLevel: ptr[float32](42.0),
	}))
	server := NewModelServer(model)
	conn := wrap.ServerToClient(soundsensorpb.SoundSensorApi_ServiceDesc, server)
	client := soundsensorpb.NewSoundSensorApiClient(conn)

	soundLevel, err := client.GetSoundLevel(ctx, &soundsensorpb.GetSoundLevelRequest{})
	if err != nil {
		t.Fatalf("GetSoundLevel failed: %v", err)
	}
	expect := &soundsensorpb.SoundLevel{SoundPressureLevel: ptr[float32](42.0)}
	if diff := cmp.Diff(expect, soundLevel, protocmp.Transform()); diff != "" {
		t.Errorf("GetSoundLevel returned unexpected value (-want +got):\n%s", diff)
	}

	stream, err := client.PullSoundLevel(ctx, &soundsensorpb.PullSoundLevelRequest{})
	if err != nil {
		t.Fatalf("PullSoundLevel failed: %v", err)
	}
	res, err := stream.Recv()
	if err != nil {
		t.Fatalf("PullSoundLevel Recv failed: %v", err)
	}
	expectRes := &soundsensorpb.PullSoundLevelResponse{Changes: []*soundsensorpb.PullSoundLevelResponse_Change{{
		SoundLevel: &soundsensorpb.SoundLevel{SoundPressureLevel: ptr[float32](42.0)},
	}}}
	diff := cmp.Diff(expectRes, res, protocmp.Transform(), protocmp.IgnoreFields(&soundsensorpb.PullSoundLevelResponse_Change{}, "change_time"))
	if diff != "" {
		t.Errorf("PullSoundLevel returned unexpected value (-want +got):\n%s", diff)
	}

	go func() {
		_, _ = model.UpdateSoundLevel(&soundsensorpb.SoundLevel{SoundPressureLevel: ptr[float32](43.0)})
	}()
	res, err = stream.Recv()
	if err != nil {
		t.Fatalf("PullSoundLevel Recv failed: %v", err)
	}
	expectRes = &soundsensorpb.PullSoundLevelResponse{Changes: []*soundsensorpb.PullSoundLevelResponse_Change{{
		SoundLevel: &soundsensorpb.SoundLevel{SoundPressureLevel: ptr[float32](43.0)},
	}}}
	diff = cmp.Diff(expectRes, res, protocmp.Transform(), protocmp.IgnoreFields(&soundsensorpb.PullSoundLevelResponse_Change{}, "change_time"))
	if diff != "" {
		t.Errorf("PullSoundLevel returned unexpected value (-want +got):\n%s", diff)
	}
}

func ptr[T any](v T) *T {
	return &v
}
