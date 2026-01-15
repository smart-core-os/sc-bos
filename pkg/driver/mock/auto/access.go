package auto

import (
	"context"
	"slices"
	"time"

	"golang.org/x/exp/rand"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/accesspb"
	gen_accesspb "github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/actorpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/maps"
)

func Access(model *accesspb.Model) service.Lifecycle {
	grants := maps.Values(gen_accesspb.AccessAttempt_Grant_value)
	slices.Sort(grants)
	grants = grants[1:]
	reasons := []string{
		"It's Monday, everyone can come in",
		"Unknown card ID",
		"",
		"Card expired",
	}
	actors := []*actorpb.Actor{
		nil,
		{DisplayName: "Scott Lang", Ids: map[string]string{"card": "1234567890"}},
		{DisplayName: "Hope Van Dyne", Ids: map[string]string{"card": "0987654321"}},
		{DisplayName: "Janet Van Dyne", Ids: map[string]string{"card": "1234567890"}},
	}

	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer((30 * time.Second) + time.Duration(rand.Float32())*time.Minute)
			for {
				state := &gen_accesspb.AccessAttempt{
					Grant:  gen_accesspb.AccessAttempt_Grant(grants[rand.Intn(len(grants))]),
					Reason: reasons[rand.Intn(len(reasons))],
					Actor:  actors[rand.Intn(len(actors))],
				}
				_, _ = model.UpdateLastAccessAttempt(state)

				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					timer = time.NewTimer((30 * time.Second) + time.Duration(rand.Float32())*time.Minute)
				}
			}
		}()
		return nil
	}), service.WithParser(func(data []byte) (string, error) {
		return string(data), nil
	}))
	_, _ = slc.Configure([]byte{}) // call configure to ensure we load when start is called.
	return slc
}
