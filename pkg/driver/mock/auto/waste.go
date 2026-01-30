package auto

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/wastepb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func WasteRecordsAuto(model *wastepb.Model) *service.Service[string] {

	slc := service.New(service.MonoApply(func(ctx context.Context, _ string) error {
		ticker := time.NewTicker(30 * time.Second)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					_, _ = model.GenerateWasteRecord(timestamppb.Now())
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
