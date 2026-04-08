package auto

// ChannelAuto simulates a television in a meeting room or office lobby.
// During business hours the channel changes periodically among a realistic set of broadcast news
// and business channels. Outside business hours the TV is left on the first channel, as if
// left on standby or used infrequently by security staff.
// Updates every 30–120 minutes, representing channel changes between or during meetings.

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/scale"
	"github.com/smart-core-os/sc-bos/pkg/proto/channelpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// officeChannels represents a realistic set of channels found on a meeting room or lobby TV.
var officeChannels = []*channelpb.Channel{
	{Id: "bbc-news", ChannelNumber: "101", Title: "BBC News"},
	{Id: "cnn-intl", ChannelNumber: "102", Title: "CNN International"},
	{Id: "bloomberg", ChannelNumber: "103", Title: "Bloomberg TV"},
	{Id: "sky-news", ChannelNumber: "104", Title: "Sky News"},
	{Id: "natgeo", ChannelNumber: "105", Title: "National Geographic"},
}

// ChannelAuto returns a Lifecycle that drives a Channel model to simulate a meeting room or lobby TV.
// During business hours a channel is chosen at random from a set of news/business channels;
// otherwise the default channel (BBC News) is shown.
func ChannelAuto(model *channelpb.Model) service.Lifecycle {
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(30*time.Minute, 120*time.Minute))
			defer timer.Stop()
			update := func() {
				factor := scale.NineToFive.Now()
				ch := officeChannels[0] // BBC News — standby default
				if factor > 0.3 {
					ch = oneOf(officeChannels...)
				}
				_, _ = model.UpdateChosenChannel(ch)
			}
			update()
			for {
				timer.Reset(durationBetween(30*time.Minute, 120*time.Minute))
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					update()
				}
			}
		}()
		return nil
	}), service.WithParser(func(data []byte) (string, error) {
		return string(data), nil
	}))
	_, _ = slc.Configure([]byte{})
	return slc
}
