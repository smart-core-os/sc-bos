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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/statuspb"
	"github.com/vanti-dev/sc-bos/pkg/minibus"

	"github.com/vanti-dev/sc-bos/pkg/driver/hikcentral/api"
	"github.com/vanti-dev/sc-bos/pkg/driver/hikcentral/config"
)

type Camera struct {
	traits.UnimplementedPtzApiServer
	gen.UnimplementedMqttServiceServer
	gen.UnimplementedUdmiServiceServer
	gen.UnimplementedStatusApiServer

	client *api.Client
	logger *zap.Logger
	Now    func() time.Time

	conf *config.Camera

	lock  sync.Mutex
	state *CameraState
	bus   minibus.Bus[*CameraState]
}

func NewCamera(client *api.Client, logger *zap.Logger, conf *config.Camera) *Camera {
	return &Camera{
		client: client,
		logger: logger,
		conf:   conf,
		state:  &CameraState{},
	}
}

func (c *Camera) UpdatePtz(ctx context.Context, request *traits.UpdatePtzRequest) (*traits.Ptz, error) {
	if request.State == nil {
		return nil, status.Error(codes.InvalidArgument, "no PTZ state in request")
	}

	if request.State.Preset != "" {
		i, err := strconv.Atoi(request.State.Preset)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid preset, [1,256]")
		}
		_, err = c.client.CameraPtzControl(ctx, &api.PtzRequest{
			CameraIndexCode: c.conf.IndexCode,
			Action:          1, // stop
			Command:         "GOTO_PRESET",
			PresetIndex:     i,
		})
		if err != nil {
			c.logger.Warn("error going to preset", zap.Int("preset", i), zap.String("error", err.Error()))
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
		_, err := c.client.CameraPtzControl(ctx, &api.PtzRequest{
			CameraIndexCode: c.conf.IndexCode,
			Action:          1, // stop
			Command:         cmd,
		})
		if err != nil {
			c.logger.Warn("error controlling PTZ", zap.String("command", cmd), zap.String("error", err.Error()))
			return nil, status.Errorf(codes.Unknown, "error controlling PTZ: %s", err.Error())
		}

		return nil, nil
	}

	return nil, nil
}

func (c *Camera) Stop(ctx context.Context, _ *traits.StopPtzRequest) (*traits.Ptz, error) {
	wg := sync.WaitGroup{}

	// we don't know which command(s) are running, so stop them all!

	var multiErr error
	for _, command := range api.Commands {
		command := command // save for goroutine usage
		wg.Add(1)
		go func() {
			_, err := c.client.CameraPtzControl(ctx, &api.PtzRequest{
				CameraIndexCode: c.conf.IndexCode,
				Action:          1, // stop
				Command:         command,
			})
			if err != nil {
				c.logger.Warn("error stopping PTZ", zap.String("command", command), zap.String("error", err.Error()))
				err = multierr.Combine(multiErr, err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return nil, multiErr
}

func (c *Camera) PullMessages(_ *gen.PullMessagesRequest, server gen.MqttService_PullMessagesServer) error {
	changes := c.bus.Listen(server.Context())

	for change := range changes {
		asJson, err := json.Marshal(change)
		if err != nil {
			c.logger.Warn("unable to marshal message as JSON", zap.String("error", err.Error()), zap.Any("change", change))
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
			c.logger.Warn("unable to marshal message as JSON", zap.String("error", err.Error()), zap.Any("change", change))
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

func (c *Camera) GetCurrentStatus(_ context.Context, _ *gen.GetCurrentStatusRequest) (*gen.StatusLog, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return stateToStatusLog(c.state), nil
}

func (c *Camera) PullCurrentStatus(request *gen.PullCurrentStatusRequest, server gen.StatusApi_PullCurrentStatusServer) error {
	for state := range c.bus.Listen(server.Context()) {
		sl := stateToStatusLog(state)
		msg := &gen.PullCurrentStatusResponse{Changes: []*gen.PullCurrentStatusResponse_Change{
			{Name: request.Name, CurrentStatus: sl},
		}}
		if err := server.Send(msg); err != nil {
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

func stateToStatusLog(state *CameraState) *gen.StatusLog {
	problems := statuspb.ProblemMerger{}
	if !state.CamState {
		problems.AddProblem(&gen.StatusLog_Problem{
			Name:        "state",
			Level:       gen.StatusLog_NON_FUNCTIONAL,
			Description: "Camera is inactive",
			RecordTime:  timestamppb.New(state.CamStateTime),
		})
	}
	if state.CamFlt {
		problems.AddProblem(&gen.StatusLog_Problem{
			Name:        "event",
			Level:       gen.StatusLog_NOTICE,
			Description: "Camera recently reported a fault",
			RecordTime:  timestamppb.New(state.CamFltTime),
		})
	}
	return problems.Build()
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
		res, err := c.client.ListEvents(ctx, &api.EventsRequest{
			EventTypes: strings.Join([]string{
				api.VideoLossAlarm,
				api.VideoTamperingAlarm,
				api.CameraRecordingExceptionAlarm,
				api.CameraRecordingRecoveredAlarm,
			}, ","),
			SrcType:    "camera",
			SrcIndexes: c.conf.IndexCode,
			StartTime:  formatTime(start),
			EndTime:    formatTime(end),
			Request: api.Request{
				PageNo:   pageNum,
				PageSize: pageSize,
			},
		})
		if err != nil {
			logger.Warn("response error", zap.String("error", err.Error()))
		} else {
			var videoLoss, videoTamper, recordingException, recordingRecovered bool
			var events []api.EventRecord
			for _, record := range res.List {
				if record.StopTime != "" {
					continue // this alarm is done
				}
				if record.LinkCameraIndexCode != c.conf.IndexCode {
					continue // not for this camera
				}
				switch record.EventType {
				case api.VideoLossAlarm:
					videoLoss = true
					events = append(events, record)
				case api.VideoTamperingAlarm:
					videoTamper = true
					events = append(events, record)
				case api.CameraRecordingExceptionAlarm:
					recordingException = true
					events = append(events, record)
				case api.CameraRecordingRecoveredAlarm:
					recordingRecovered = true
					events = append(events, record)
				}
			}
			fault := videoLoss || videoTamper || (recordingException && !recordingRecovered)
			c.updateFault(ctx, fault)
			if len(res.List) < pageSize {
				// no more pages, exit
				break
			} else {
				pageNum++
			}
		}
	}
}
func (c *Camera) getOcc(ctx context.Context) {
	now := time.Now()
	start := now.Truncate(time.Hour)
	end := start.Add(time.Hour)
	logger := c.logger.With(
		zap.String("method", "getOcc"),
		zap.String("startTime", formatTime(start)), zap.String("endTime", formatTime(end)),
	)
	pageNum := 1
	pageSize := 100
	for {
		res, err := c.client.GetCameraPeopleStats(ctx, &api.StatsRequest{
			CameraIndexCodes: c.conf.IndexCode,
			StatisticsType:   api.StatisticsTypeByHour,
			StartTime:        formatTime(start),
			EndTime:          formatTime(end),
			Request: api.Request{
				PageNo:   pageNum,
				PageSize: pageSize,
			},
		})
		if err != nil {
			logger.Warn("response error", zap.String("error", err.Error()))
		} else {
			if len(res.List) == 0 {
				logger.Warn("no people count data in response", zap.Any("res", res))
			} else if res.List[0].CameraIndexCode != c.conf.IndexCode {
				continue // not for this camera
			} else {
				i := res.List[0]
				count := i.EnterNum - i.ExitNum
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
	res, err := c.client.GetCameraPreviewUrl(ctx, &api.CameraPreviewRequest{CameraRequest: api.CameraRequest{CameraIndexCode: c.conf.IndexCode}})
	if err != nil {
		logger.Warn("response error", zap.String("error", err.Error()))
	} else {
		bytes, err := json.Marshal(res)
		if err != nil {
			logger.Warn("error serialising stream info", zap.String("error", err.Error()))
		} else {
			c.updateVideo(ctx, string(bytes))
		}
	}
}

func (c *Camera) getInfo(ctx context.Context) {
	logger := c.logger.With(zap.String("method", "getInfo"))
	res, err := c.client.GetCameraInfo(ctx, &api.CameraRequest{CameraIndexCode: c.conf.IndexCode})
	if err != nil {
		logger.Warn("response error", zap.String("error", err.Error()))
	} else {
		active := res.Status == api.CameraStatusOnline
		c.updateActive(ctx, active)
	}
}

func (c *Camera) updateCount(ctx context.Context, count string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.state.CamOcc = count
	c.bus.Send(ctx, c.state)
}

func (c *Camera) updateVideo(ctx context.Context, video string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.state.CamVideo = video
	c.bus.Send(ctx, c.state)
}

func (c *Camera) updateFault(ctx context.Context, fault bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.state.CamFlt = fault
	c.state.CamFltTime = c.now()
	c.bus.Send(ctx, c.state)
}

func (c *Camera) updateActive(ctx context.Context, active bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.state.CamState = active
	c.state.CamStateTime = c.now()
	c.bus.Send(ctx, c.state)
}

func (c *Camera) updateState(ctx context.Context, new *CameraState) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.state = new
	c.bus.Send(ctx, c.state)
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

func parseTime(s string) (time.Time, error) {
	return time.Parse(RFC3339NumericZone, s)
}
