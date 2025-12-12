package hikcentral

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/api"
	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/config"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
)

type Camera struct {
	traits.UnimplementedPtzApiServer
	gen.UnimplementedMqttServiceServer
	gen.UnimplementedUdmiServiceServer

	client *client
	logger *zap.Logger
	Now    func() time.Time

	conf *config.Camera

	lock       sync.Mutex
	state      *CameraState
	bus        minibus.Bus[*CameraState]
	faultCheck *healthpb.FaultCheck
}

func NewCamera(client *client, logger *zap.Logger, conf *config.Camera, fc *healthpb.FaultCheck) *Camera {
	return &Camera{
		client:     client,
		conf:       conf,
		faultCheck: fc,
		logger:     logger,
		state:      &CameraState{},
	}
}

func (c *Camera) UpdatePtz(ctx context.Context, request *traits.UpdatePtzRequest) (*traits.Ptz, error) {
	if request.State == nil {
		return nil, status.Error(codes.InvalidArgument, "no PTZ state in request")
	}

	if request.State.Preset != "" {
		i, err := strconv.Atoi(request.State.Preset)
		if err != nil || i < 1 || i > 256 {
			return nil, status.Error(codes.InvalidArgument, "invalid preset, [1,256]")
		}
		_, err = c.client.cameraPtzControl(ctx, &api.PtzRequest{
			CameraIndexCode: c.conf.IndexCode,
			Action:          1, // stop
			Command:         "GOTO_PRESET",
			PresetIndex:     i,
		}, c.faultCheck)
		if err != nil {
			c.logger.Warn("error going to preset", zap.Int("preset", i), zap.Error(err))
			return nil, status.Errorf(codes.Unknown, "error going to preset: %s", err.Error())
		}
		return nil, nil
	}

	if request.State.Movement != nil {
		mov := request.State.Movement
		if mov.Direction == nil {
			return nil, status.Error(codes.InvalidArgument, "no direction specified")
		}
		if mov.Direction.Pan == 0 && mov.Direction.Tilt == 0 && mov.Direction.Zoom == 0 {
			return nil, status.Error(codes.InvalidArgument, "no direction specified")
		}
		speed := mov.Speed
		if speed == 0 {
			speed = 40 // default
		}
		if speed > 60 || speed < 20 {
			return nil, status.Error(codes.InvalidArgument, "invalid speed, [20,60]")
		}
		cmd := api.MovementToCommand(mov)
		_, err := c.client.cameraPtzControl(ctx, &api.PtzRequest{
			CameraIndexCode: c.conf.IndexCode,
			Action:          1, // stop
			Command:         cmd,
		}, c.faultCheck)
		if err != nil {
			c.logger.Warn("error controlling PTZ", zap.String("command", cmd), zap.Error(err))
			return nil, status.Errorf(codes.Unknown, "error controlling PTZ: %s", err.Error())
		}

		return nil, nil
	}

	return nil, nil
}

func (c *Camera) Stop(ctx context.Context, _ *traits.StopPtzRequest) (*traits.Ptz, error) {
	wg, ctx := errgroup.WithContext(ctx)

	// we don't know which command(s) are running, so stop them all!

	mu := sync.Mutex{}
	var multiErr error
	for _, command := range api.Commands {
		command := command // save for goroutine usage
		wg.Go(func() error {
			_, err := c.client.cameraPtzControl(ctx, &api.PtzRequest{
				CameraIndexCode: c.conf.IndexCode,
				Action:          1, // stop
				Command:         command,
			}, c.faultCheck)
			if err != nil {
				c.logger.Warn("error stopping PTZ", zap.String("command", command), zap.Error(err))
				mu.Lock()
				multiErr = multierr.Combine(multiErr, err)
				mu.Unlock()
			}
			return nil
		})
	}
	_ = wg.Wait()
	return nil, multiErr
}

func (c *Camera) PullMessages(_ *gen.PullMessagesRequest, server gen.MqttService_PullMessagesServer) error {
	changes := c.bus.Listen(server.Context())

	for change := range changes {
		asJson, err := json.Marshal(change)
		if err != nil {
			c.logger.Warn("unable to marshal message as JSON", zap.Error(err), zap.Any("change", change))
			continue
		}
		msg := &gen.PullMessagesResponse{
			Name:    c.conf.Name,
			Topic:   c.conf.Topic,
			Payload: string(asJson),
		}
		err = server.Send(msg)
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}

func (c *Camera) PullControlTopics(request *gen.PullControlTopicsRequest, server gen.UdmiService_PullControlTopicsServer) error {
	return c.UnimplementedUdmiServiceServer.PullControlTopics(request, server)
}

func (c *Camera) OnMessage(ctx context.Context, request *gen.OnMessageRequest) (*gen.OnMessageResponse, error) {
	return c.UnimplementedUdmiServiceServer.OnMessage(ctx, request)
}

func (c *Camera) PullExportMessages(_ *gen.PullExportMessagesRequest, server gen.UdmiService_PullExportMessagesServer) error {
	changes := c.bus.Listen(server.Context())

	for change := range changes {
		asJson, err := marshalUDMIPayload(change)
		if err != nil {
			c.logger.Warn("unable to marshal message as JSON", zap.Error(err), zap.Any("change", change))
			continue
		}
		msg := &gen.PullExportMessagesResponse{
			Name: c.conf.Name,
			Message: &gen.MqttMessage{
				Topic:   c.conf.Topic + "/event/pointset/points",
				Payload: string(asJson),
			},
		}
		err = server.Send(msg)
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}

func marshalUDMIPayload(msg any) ([]byte, error) {
	type val struct {
		PresentValue any `json:"present_value"`
	}
	out := make(map[string]val)
	mt := reflect.TypeOf(msg)
	if mt.Kind() == reflect.Ptr {
		mt = mt.Elem()
	}
	mv := reflect.ValueOf(msg)
	if mv.Kind() == reflect.Ptr {
		mv = mv.Elem()
	}
	for i := 0; i < mt.NumField(); i++ {
		field := mt.Field(i)
		key := field.Name
		var omitEmpty bool
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			if p, _, ok := strings.Cut(jsonTag, ","); ok {
				key = p
			}
			omitEmpty = strings.Contains(jsonTag, ",omitempty")
		}

		value := mv.Field(i)
		if value.IsZero() && omitEmpty {
			continue
		}
		out[key] = val{PresentValue: value.Interface()}
	}

	bs, err := json.Marshal(out)
	return bs, err
}

func (c *Camera) getEvents(ctx context.Context) {
	now := c.now()
	start := now.Truncate(time.Hour)
	end := start.Add(time.Hour)
	logger := c.logger.With(zap.String("method", "getEvents"),
		zap.String("startTime", formatTime(start)), zap.String("endTime", formatTime(end)))

	pageNum := 1
	pageSize := 100
	for {
		res, err := c.client.listEvents(ctx, &api.EventsRequest{
			EventTypes: strings.Join([]string{
				api.VideoLossAlarm,
				api.VideoTamperingAlarm,
				api.CameraRecordingExceptionAlarm,
				api.CameraRecordingRecovered,
			}, ","),
			SrcType:    "camera",
			SrcIndexes: c.conf.IndexCode,
			StartTime:  formatTime(start),
			EndTime:    formatTime(end),
			Request: api.Request{
				PageNo:   pageNum,
				PageSize: pageSize,
			},
		}, c.faultCheck)
		if err != nil {
			logger.Warn("response error", zap.Error(err))
			break
		} else {
			c.processEventRecords(ctx, res.List)
			if len(res.List) < pageSize {
				// no more pages, exit
				break
			} else {
				pageNum++
			}
		}
	}
}

type allFaults map[string]bool

func (f allFaults) hasFault() bool {
	return f[api.VideoLossAlarm] || f[api.VideoTamperingAlarm] || f[api.CameraRecordingExceptionAlarm]
}

func (c *Camera) processEventRecords(ctx context.Context, records []api.EventRecord) {
	faults := make(allFaults)
	clearRecordingException := false
	for _, record := range records {
		if record.StopTime != "" {
			continue // this alarm is done
		}
		if record.LinkCameraIndexCode != c.conf.IndexCode {
			continue // not for this camera
		}
		switch record.EventType {
		case api.VideoLossAlarm:
			faults[api.VideoLossAlarm] = true
		case api.VideoTamperingAlarm:
			faults[api.VideoTamperingAlarm] = true
		case api.CameraRecordingExceptionAlarm:
			faults[api.CameraRecordingExceptionAlarm] = true
		case api.CameraRecordingRecovered:
			// if we detect a recording-recovered alarm,
			// we want to clear the recording exception fault
			clearRecordingException = true
		}
	}
	if clearRecordingException {
		faults[api.CameraRecordingExceptionAlarm] = false
	}
	updateDeviceFaults(faults, c.faultCheck)
	fault := faults.hasFault()
	c.updateFault(ctx, fault)
}

func (c *Camera) getOcc(ctx context.Context) {
	now := c.now()
	start := now.Truncate(time.Hour)
	end := start.Add(time.Hour)
	logger := c.logger.With(
		zap.String("method", "getOcc"),
		zap.String("startTime", formatTime(start)), zap.String("endTime", formatTime(end)),
	)
	pageNum := 1
	pageSize := 100
	for {
		res, err := c.client.getCameraPeopleStats(ctx, &api.StatsRequest{
			CameraIndexCodes: c.conf.IndexCode,
			StatisticsType:   api.StatisticsTypeByHour,
			StartTime:        formatTime(start),
			EndTime:          formatTime(end),
			Request: api.Request{
				PageNo:   pageNum,
				PageSize: pageSize,
			},
		}, c.faultCheck)
		if err != nil {
			logger.Warn("response error", zap.Error(err))
			break
		} else {
			if len(res.List) == 0 {
				logger.Warn("no people count data in response", zap.Any("res", res))
			} else if res.List[0].CameraIndexCode != c.conf.IndexCode {
				continue // not for this camera
			} else {
				i := res.List[0]
				count := i.EnterNum - i.ExitNum
				if count < 0 {
					count = 0
				}
				c.updateCount(ctx, strconv.Itoa(count))
			}
			if len(res.List) < pageSize {
				// no more pages, exit
				break
			} else {
				pageNum++
			}
		}
	}
}

func (c *Camera) getStream(ctx context.Context) {
	logger := c.logger.With(zap.String("method", "getStream"))
	res, err := c.client.getCameraPreviewUrl(ctx, &api.CameraPreviewRequest{CameraRequest: api.CameraRequest{CameraIndexCode: c.conf.IndexCode}}, c.faultCheck)
	if err != nil {
		logger.Warn("response error", zap.Error(err))
	} else {
		bytes, err := json.Marshal(res)
		if err != nil {
			logger.Warn("error serialising stream info", zap.Error(err))
		} else {
			c.updateVideo(ctx, string(bytes))
		}
	}
}

func (c *Camera) getInfo(ctx context.Context) {
	logger := c.logger.With(zap.String("method", "getInfo"))
	res, err := c.client.getCameraInfo(ctx, &api.CameraRequest{CameraIndexCode: c.conf.IndexCode}, c.faultCheck)
	if err != nil {
		logger.Warn("response error", zap.Error(err))
	} else {
		active := res.Status == api.CameraStatusOnline
		c.updateActive(ctx, active)
	}
}

func (c *Camera) updateCount(ctx context.Context, count string) {
	c.updateAndNotify(ctx, func() {
		c.state.CamOcc = count
	})
}

func (c *Camera) updateVideo(ctx context.Context, video string) {
	c.updateAndNotify(ctx, func() {
		c.state.CamVideo = video
	})
}

func (c *Camera) updateFault(ctx context.Context, fault bool) {
	c.updateAndNotify(ctx, func() {
		c.state.CamFlt = fault
		c.state.CamFltTime = c.now()
	})
}

func (c *Camera) updateActive(ctx context.Context, active bool) {
	c.updateAndNotify(ctx, func() {
		c.state.CamState = active
		c.state.CamStateTime = c.now()
	})
}

func (c *Camera) updateState(ctx context.Context, new *CameraState) {
	c.updateAndNotify(ctx, func() {
		c.state = new
	})
}

// updateAndNotify safely updates the camera state and notifies listeners.
// The updateFn is called while holding the lock, then a copy is made and sent to the bus.
func (c *Camera) updateAndNotify(ctx context.Context, updateFn func()) {
	c.lock.Lock()
	updateFn()
	stateCopy := c.copyState()
	c.lock.Unlock()
	c.bus.Send(ctx, stateCopy)
}

func (c *Camera) copyState() *CameraState {
	stateCopy := *c.state
	if c.state.CamAim != nil {
		ptzCopy := *c.state.CamAim
		stateCopy.CamAim = &ptzCopy
	}
	return &stateCopy
}

func (c *Camera) now() time.Time {
	if c.Now == nil {
		return time.Now()
	}
	return c.Now()
}

type CameraState struct {
	CamState bool   `json:"camState"`
	CamFlt   bool   `json:"camFlt"`
	CamAim   *PTZ   `json:"camAim,omitempty"`
	CamOcc   string `json:"camOcc,omitempty"`
	CamVideo string `json:"camVideo,omitempty"`

	CamStateTime time.Time `json:"-"`
	CamFltTime   time.Time `json:"-"`
}

func (c *CameraState) IsEqual(c2 *CameraState) bool {
	return c.CamState == c2.CamState &&
		c.CamFlt == c2.CamFlt &&
		c.CamOcc == c2.CamOcc &&
		c.CamVideo == c2.CamVideo &&
		(c.CamAim == c2.CamAim || c.CamAim != nil && c.CamAim.IsEqual(c2.CamAim))
}

type PTZ struct {
	Pan  string `json:"pan,omitempty"`
	Tilt string `json:"tilt,omitempty"`
	Zoom string `json:"zoom,omitempty"`
}

func (p *PTZ) IsEqual(p2 *PTZ) bool {
	if p == p2 {
		return true
	}
	if p2 == nil {
		return false
	}
	return p.Pan == p2.Pan &&
		p.Tilt == p2.Tilt &&
		p.Zoom == p2.Zoom
}

const RFC3339NumericZone = "2006-01-02T15:04:05-07:00"

func formatTime(t time.Time) string {
	return t.Format(RFC3339NumericZone)
}
