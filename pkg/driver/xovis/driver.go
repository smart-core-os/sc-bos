package xovis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/xovis/config"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/node"
	gen_udmipb "github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/enterleavesensorpb"
	"github.com/smart-core-os/sc-golang/pkg/trait/occupancysensorpb"
)

const DriverName = "xovis"

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {

	d := &Driver{
		announcer:   node.NewReplaceAnnouncer(services.Node),
		health:      services.Health,
		httpMux:     services.HTTPMux,
		logger:      services.Logger.Named(DriverName),
		pushDataBus: &minibus.Bus[PushData]{},
	}
	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithParser(config.ParseConfig),
	)
	return d
}

type Driver struct {
	*service.Service[config.Root]
	announcer   *node.ReplaceAnnouncer
	health      *healthpb.Checks
	httpMux     *http.ServeMux
	logger      *zap.Logger
	pushDataBus *minibus.Bus[PushData]

	config      config.Root
	client      *client
	server      *http.Server // only used if httpPort is configured for the webhook
	udmiServers []*udmiServiceServer
}

func (d *Driver) applyConfig(ctx context.Context, conf config.Root) error {
	announcer := d.announcer.Replace(ctx)
	grp, ctx := errgroup.WithContext(ctx)

	// A route can't be removed from an HTTP ServeMux, so if it's been changed or removed then we can't support the
	// new configuration. This is likely to be rare in practice. Adding a route is fine.
	var oldWebhook, newWebhook string
	if d.config.DataPush != nil {
		oldWebhook = d.config.DataPush.WebhookPath
	}
	if conf.DataPush != nil {
		newWebhook = conf.DataPush.WebhookPath
	}
	if oldWebhook != "" && newWebhook != oldWebhook {
		return errors.New("can't change webhook path once service is running")
	}

	// create a new client to communicate with the Xovis sensor
	pass, err := conf.LoadPassword()
	if err != nil {
		return err
	}
	d.client = newInsecureClient(conf.Host, conf.Username, pass)

	// announce new devices
	for _, dev := range conf.Devices {
		features := []node.Feature{node.HasMetadata(dev.Metadata)}

		faultCheck, err := d.health.NewFaultCheck(dev.Name, commsHealthCheck)
		if err != nil {
			d.logger.Error("failed to create health check", zap.String("device", dev.Name), zap.Error(err))
			return err
		}

		var occupancyVal *resource.Value
		if dev.Occupancy != nil {
			occupancy := &occupancyServer{
				client:         d.client,
				faultCheck:     faultCheck,
				multiSensor:    conf.MultiSensor,
				logicID:        dev.Occupancy.ID,
				bus:            d.pushDataBus,
				OccupancyTotal: resource.NewValue(resource.WithInitialValue(&traits.Occupancy{}), resource.WithNoDuplicates()),
			}
			features = append(features, node.HasTrait(trait.OccupancySensor,
				node.WithClients(occupancysensorpb.WrapApi(occupancy))))
			occupancyVal = occupancy.OccupancyTotal
		}
		var enterLeaveVal *resource.Value
		if dev.EnterLeave != nil {
			enterLeave := &enterLeaveServer{
				client:          d.client,
				faultCheck:      faultCheck,
				logicID:         dev.EnterLeave.ID,
				multiSensor:     conf.MultiSensor,
				bus:             d.pushDataBus,
				EnterLeaveTotal: resource.NewValue(resource.WithInitialValue(&traits.EnterLeaveEvent{}), resource.WithNoDuplicates()),
			}

			features = append(features, node.HasTrait(trait.EnterLeaveSensor,
				node.WithClients(enterleavesensorpb.WrapApi(enterLeave))))
			enterLeaveVal = enterLeave.EnterLeaveTotal
		}

		if enterLeaveVal != nil || occupancyVal != nil {
			server := newUdmiServiceServer(d.logger.Named("udmiServiceServer"), enterLeaveVal, occupancyVal, dev.UDMITopicPrefix)
			d.udmiServers = append(d.udmiServers, server)
			features = append(features, node.HasTrait(udmipb.TraitName,
				node.WithClients(gen_udmipb.WrapService(server))))
		}

		announcer.Announce(dev.Name, features...)
	}
	// register data push webhook
	if d.server != nil {
		d.server.Close()
		d.server = nil
	}
	if dp := conf.DataPush; dp != nil && dp.WebhookPath != "" {
		if dp.HTTPPort > 0 {
			// setup a dedicate http server for the webhook, we use http
			lis, err := net.Listen("tcp", fmt.Sprintf(":%d", dp.HTTPPort))
			if err != nil {
				return err
			}
			mux := http.NewServeMux()
			mux.HandleFunc(dp.WebhookPath, d.handleWebhook)
			d.server = &http.Server{
				Handler: mux,
			}
			grp.Go(func() error {
				return d.server.Serve(lis)
			})
		} else {
			d.httpMux.HandleFunc(dp.WebhookPath, d.handleWebhook)
		}
	}

	d.config = conf
	go func() {
		err = grp.Wait()
		if err != nil {
			d.logger.Error("xovis driver stopped unexpectedly", zap.Error(err))
		}
		if d.server != nil {
			_ = d.server.Close()
		}
		d.client.Client.CloseIdleConnections()
	}()

	return nil
}

func (d *Driver) handleWebhook(response http.ResponseWriter, request *http.Request) {
	// verify HTTP method
	if request.Method != http.MethodPost {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// verify request body is JSON
	mediatype, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediatype != "application/json" {
		response.WriteHeader(http.StatusUnsupportedMediaType)
		_, _ = response.Write([]byte("invalid content type"))
		return
	}

	// read request body and parse
	rawBody, err := io.ReadAll(http.MaxBytesReader(response, request.Body, 10*1024*1024))
	if err != nil {
		maxBytesErr := &http.MaxBytesError{}
		if errors.As(err, &maxBytesErr) {
			response.WriteHeader(http.StatusRequestEntityTooLarge)
		} else {
			// If the error was not size-related then the connection probably
			// dropped. It's unlikely the client will receive the error we send here.
			response.WriteHeader(http.StatusBadRequest)
		}
		return
	}
	var body PushData
	err = json.Unmarshal(rawBody, &body)
	if err != nil {
		d.logger.Debug("failed to parse webhook body", zap.Error(err))
		response.WriteHeader(http.StatusBadRequest)
		_, _ = response.Write([]byte(err.Error()))
		return
	}

	n := 150
	if len(rawBody) < n {
		n = len(rawBody)
	}
	d.logger.Debug("received webhook", zap.ByteString("body", rawBody[:n]))

	// send the data to the bus
	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()
	_ = d.pushDataBus.Send(ctx, body)
}
