package node

import (
	"context"
	"testing"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/trait/onoffpb"
)

func TestNode_Client(t *testing.T) {
	n := New("Test")
	srv := onoffpb.NewModelServer(onoffpb.NewModel(onoffpb.WithInitialOnOff(&traits.OnOff{State: traits.OnOff_ON})))
	n.Announce("foo",
		HasTrait(trait.OnOff),
		HasServer(traits.RegisterOnOffApiServer, traits.OnOffApiServer(srv)),
	)

	var client traits.OnOffApiClient
	err := n.Client(&client)
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	if client == nil {
		t.Fatalf("Expecting client but got none")
	}
	state, err := client.GetOnOff(context.Background(), &traits.GetOnOffRequest{Name: "foo"})
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	if state.State != traits.OnOff_ON {
		t.Fatalf("State doesn't match, want %v, got %v", traits.OnOff_ON, state.State)
	}
}
