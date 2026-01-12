package auto

import (
	"context"
	"time"

	"golang.org/x/exp/rand"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	gen_statuspb "github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func Status(model *statuspb.Model, name string) service.Lifecycle {
	levels := []gen_statuspb.StatusLog_Level{
		gen_statuspb.StatusLog_NOMINAL,
		gen_statuspb.StatusLog_NOTICE,
		gen_statuspb.StatusLog_REDUCED_FUNCTION,
		gen_statuspb.StatusLog_NON_FUNCTIONAL,
		gen_statuspb.StatusLog_OFFLINE,
	}
	descriptions := map[gen_statuspb.StatusLog_Level][]string{
		gen_statuspb.StatusLog_NOMINAL:          {"Connection Successful", "No issues detected", "All systems operational"},
		gen_statuspb.StatusLog_NOTICE:           {"Device is being slow", "Check your configuration"},
		gen_statuspb.StatusLog_REDUCED_FUNCTION: {"Device is not responding", "Timeout error"},
		gen_statuspb.StatusLog_NON_FUNCTIONAL:   {"Device is not connected", "No data available"},
		gen_statuspb.StatusLog_OFFLINE:          {"Device is offline", "Unable to connect"},
	}
	names := []string{name, name + ":Connection", name + ":Faults"}
	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		go func() {
			timer := time.NewTimer(durationBetween(30*time.Second, 2*time.Minute))
			for {
				level := oneOf(levels...)
				// 90% chance that the problem is nominal, to make things more likely to be working.
				if rand.Float32() < 0.9 {
					level = gen_statuspb.StatusLog_NOMINAL
				}
				problem := &gen_statuspb.StatusLog_Problem{
					Name:        oneOf(names...),
					Level:       level,
					Description: oneOf(descriptions[level]...),
				}
				_, _ = model.UpdateProblem(problem)
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					timer = time.NewTimer(durationBetween(30*time.Second, 2*time.Minute))
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
