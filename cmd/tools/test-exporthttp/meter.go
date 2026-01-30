package main

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/historypb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/history/memstore"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

// announceMeter with events in order
func announceMeter(root node.Announcer, name, unit string, sleep time.Duration, events []float32) error {
	model := meter.NewModel()

	modelInfoServer := &meter.InfoServer{
		UnimplementedMeterInfoServer: meterpb.UnimplementedMeterInfoServer{},
		MeterReading:                 &meterpb.MeterReadingSupport{UsageUnit: unit},
	}

	client := node.WithClients(meterpb.WrapApi(meter.NewModelServer(model)), meterpb.WrapInfo(modelInfoServer))
	root.Announce(name, node.HasTrait(meter.TraitName, client))

	store := memstore.New()

	for _, event := range events {
		rec, err := proto.Marshal(&meterpb.MeterReading{
			Usage:     event,
			EndTime:   timestamppb.Now(),
			StartTime: timestamppb.Now(),
		})
		if err != nil {
			return err
		}
		_, err = store.Append(nil, rec)
		if err != nil {
			return err
		}
		time.Sleep(sleep)
	}

	root.Announce(name, node.HasClient(meterpb.WrapHistory(historypb.NewMeterServer(store))))

	return nil
}
