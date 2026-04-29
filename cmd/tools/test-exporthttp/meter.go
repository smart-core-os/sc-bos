package main

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/history/memstore"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/historypb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

// announceMeter with events in order
func announceMeter(root node.Announcer, name, unit string, sleep time.Duration, events []float32) error {
	model := meterpb.NewModel()

	modelInfoServer := &meterpb.InfoServer{
		UnimplementedMeterInfoServer: meterpb.UnimplementedMeterInfoServer{},
		MeterReading:                 &meterpb.MeterReadingSupport{UsageUnit: unit},
	}

	root.Announce(name,
		node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(model))),
		node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(modelInfoServer)),
		node.HasTrait(meterpb.TraitName))

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
		_, err = store.Append(context.TODO(), rec)
		if err != nil {
			return err
		}
		time.Sleep(sleep)
	}

	root.Announce(name, node.HasServer(meterpb.RegisterMeterHistoryServer, meterpb.MeterHistoryServer(historypb.NewMeterServer(store))))

	return nil
}
