package source

import (
	"context"
	"sort"
	"strings"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/smart-core-os/gobacnet/property"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/sc-bos/pkg/auto/export/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/adapt"
	dconfig "github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/rpc"
	"github.com/smart-core-os/sc-bos/pkg/task"
)

func NewBacnet(services Services) task.Starter {
	r := &bacnet{services: services}
	r.Lifecycle = task.NewLifecycle(r.applyConfig)
	r.Logger = services.Logger.Named("bacnet-driver")
	return r
}

type bacnet struct {
	*task.Lifecycle[config.BacnetSource]
	services Services
}

func (b *bacnet) applyConfig(ctx context.Context, cfg config.BacnetSource) error {
	bacnetDriverClient := rpc.NewBacnetDriverServiceClient(b.services.Node.ClientConn())

	delay := 5 * time.Second
	if cfg.COV != nil && cfg.COV.PollDelay.Duration != 0 {
		delay = cfg.COV.PollDelay.Duration
	}
	go func() {
		tick := time.NewTicker(delay)
		defer tick.Stop()

		sent := allowDuplicates()
		if cfg.Duplicates.TrackDuplicates() {
			sent = trackDuplicates(cfg.Duplicates.Cmp())
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				t := timing{}
				ctx := context.WithValue(ctx, timingKey, t)
				totalDone := t.time("total")
				err := b.publishAll(ctx, cfg, bacnetDriverClient, sent)
				if err != nil {
					b.Logger.Warn("Errors publishing changes", zap.Error(err))
				}
				totalDone()
				if cfg.PrintTiming {
					b.Logger.Debug("export complete", t.fields()...)
				}
			}
		}
	}()

	return nil
}

func (b *bacnet) publishAll(ctx context.Context, cfg config.BacnetSource, client rpc.BacnetDriverServiceClient, sent *duplicates) error {
	t, ok := ctx.Value(timingKey).(timing)
	if !ok {
		t = timing{} // will be thrown away
	}
	var allErrs error
	for _, device := range cfg.Devices {
		readDone := t.time("read")
		request, err := b.deviceToReadRequest(ctx, device, client)
		if err != nil {
			allErrs = multierr.Append(allErrs, err)
			continue
		}
		response, err := client.ReadPropertyMultiple(ctx, request)
		if err != nil {
			allErrs = multierr.Append(allErrs, err)
			continue
		}
		readDone()

		publishDone := t.time("publish")
		err = b.publishResults(ctx, device.Name, response, sent)
		if err != nil {
			allErrs = multierr.Append(allErrs, err)
			continue
		}
		publishDone()
	}
	return allErrs
}

func (b *bacnet) deviceToReadRequest(ctx context.Context, device config.BacnetDevice, client rpc.BacnetDriverServiceClient) (*rpc.ReadPropertyMultipleRequest, error) {
	readRequest := &rpc.ReadPropertyMultipleRequest{Name: device.Name}
	if len(device.Objects) == 0 {
		// read all objects from the server
		listObjectsResponse, err := client.ListObjects(ctx, &rpc.ListObjectsRequest{Name: device.Name})
		if err != nil {
			return nil, err
		}
		for _, object := range listObjectsResponse.Objects {
			// this defaults to PresentValue already
			readRequest.ReadSpecifications = append(readRequest.ReadSpecifications, &rpc.ReadPropertyMultipleRequest_ReadSpecification{
				ObjectIdentifier: object,
			})
		}
		return readRequest, nil
	}

	for _, object := range device.Objects {
		spec := &rpc.ReadPropertyMultipleRequest_ReadSpecification{
			ObjectIdentifier: adapt.ObjectIDToProto(bactypes.ObjectID(object.ID)),
		}
		for _, prop := range object.Properties {
			// todo: support array access
			spec.PropertyReferences = append(spec.PropertyReferences, &rpc.PropertyReference{
				Identifier: uint32(prop),
			})
		}
		readRequest.ReadSpecifications = append(readRequest.ReadSpecifications, spec)
	}

	return readRequest, nil
}

func (b *bacnet) publishResults(ctx context.Context, topicPrefix string, response *rpc.ReadPropertyMultipleResponse, sent *duplicates) error {
	var allErrs error
	for _, result := range response.ReadResults {
		objId := adapt.ObjectIDFromProto(result.ObjectIdentifier)
		for _, readResult := range result.Results {
			propId := property.ID(readResult.PropertyReference.Identifier)
			topicParts := []string{
				topicPrefix,
				"obj", dconfig.ObjectID(objId).String(),
				"prop", propId.String(),
			}
			topic := strings.Join(topicParts, "/")

			if commit, publish := sent.Changed(topic, readResult.Value); publish {
				data, err := protojson.Marshal(readResult.Value)
				if err != nil {
					allErrs = multierr.Append(allErrs, err)
					continue
				}
				err = b.services.Publisher.Publish(ctx, topic, string(data))
				if err != nil {
					allErrs = multierr.Append(allErrs, err)
					continue
				}
				commit()
			}
		}
	}
	return allErrs
}

type timing map[string]time.Duration
type contextKey int

var timingKey contextKey = 1

func (t timing) time(name string) func() {
	t0 := time.Now()
	return func() {
		if elapsed, ok := t[name]; ok {
			t[name] = elapsed + time.Since(t0)
		} else {
			t[name] = time.Since(t0)
		}
	}
}

func (t timing) fields() []zap.Field {
	fields := make([]zap.Field, 0, len(t))
	for name, duration := range t {
		fields = append(fields, zap.Duration(name, duration))
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Key < fields[j].Key
	})
	return fields
}
