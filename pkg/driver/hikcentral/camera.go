package hikcentral

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/api"
	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/config"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/mqttpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/ptzpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type Camera struct {
	ptzpb.UnimplementedPtzApiServer
	mqttpb.UnimplementedMqttServiceServer
	udmipb.UnimplementedUdmiServiceServer

	client *client
	logger *zap.Logger
	Now    func() time.Time

	conf *config.Camera

	lock  sync.Mutex
	state *CameraState
	// readAt is when each poll last wrote state, which only follows a successful read.
	readAt [pollCount]time.Time
	// failed is whether each poll has failed on this camera since its last successful read.
	failed [pollCount]bool
	// readLimit is how long each poll's point stays current as judged at its last successful
	// read, which is the limit that stands once the poll has failed on this camera since.
	readLimit [pollCount]time.Duration
	// staleAfter is how long the point each poll owns stays current after readAt.
	// It's zero for a poll that isn't configured.
	staleAfter [pollCount]time.Duration
	// cycles times each poll's walk over every camera, shared by the driver's cameras. nil
	// when nothing is polling, as in tests; pollCycles' methods accept a nil receiver.
	cycles     *pollCycles
	bus        minibus.Bus[*CameraState]
	faultCheck *healthpb.FaultCheck
}

// A poll is one of the independent polls that feed a Camera's state.
type poll int

const (
	infoPoll poll = iota
	occupancyPoll
	eventsPoll
	streamPoll
	pollCount
)

// pollFields names the CameraState fields each poll owns.
var pollFields = [pollCount][]string{
	infoPoll:      {"CamState"},
	occupancyPoll: {"CamOcc"},
	eventsPoll:    {"CamFlt"},
	streamPoll:    {"CamVideo"},
}

// NewCamera returns a Camera polled as polls configures. Each point counts as current
// for twice the gap between polls of it, so a single late or failed poll doesn't make it
// stale. That gap is the poll's interval, or the time cycles says the poll takes to walk
// every camera when that is longer.
func NewCamera(client *client, logger *zap.Logger, conf *config.Camera, fc *healthpb.FaultCheck, polls *config.Settings, cycles *pollCycles) *Camera {
	var staleAfter [pollCount]time.Duration
	for p, interval := range [pollCount]*jsontypes.Duration{
		infoPoll:      polls.InfoPoll,
		occupancyPoll: polls.OccupancyPoll,
		eventsPoll:    polls.EventsPoll,
		streamPoll:    polls.StreamPoll,
	} {
		if interval != nil {
			staleAfter[p] = 2 * interval.Duration
		}
	}
	return &Camera{
		client:     client,
		conf:       conf,
		faultCheck: fc,
		logger:     logger,
		state:      &CameraState{},
		staleAfter: staleAfter,
		cycles:     cycles,
	}
}

// pollCycles times each poll's walk over the cameras.
//
// A poll visits the cameras one after another, so polls of one camera are a whole walk
// apart. On a large site, or behind a slow HikCentral server, a walk can outlast the poll's
// interval, and a point judged against the interval alone goes stale before its camera has
// even been asked again (SCB-1473).
type pollCycles struct {
	mu    sync.Mutex
	start [pollCount]time.Time     // of the walk in progress, zero between walks
	last  [pollCount]time.Duration // of the latest complete walk, zero until one completes
}

func (pc *pollCycles) begin(p poll, now time.Time) {
	if pc == nil {
		return
	}
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.start[p] = now
}

func (pc *pollCycles) end(p poll, now time.Time) {
	if pc == nil {
		return
	}
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.last[p] = now.Sub(pc.start[p])
	pc.start[p] = time.Time{}
}

// length is how long p takes to walk every camera.
//
// It's the latest complete walk, not the one in progress: a walk stalled behind failing
// requests gets slower as it goes, and counting that would keep the cameras it hasn't reached
// yet looking current for as long as it stalls. The exception is the first walk, before which
// no camera has been read, so there's nothing for its time so far to keep current.
func (pc *pollCycles) length(p poll, now time.Time) time.Duration {
	if pc == nil {
		return 0
	}
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if pc.last[p] > 0 || pc.start[p].IsZero() {
		return pc.last[p]
	}
	return now.Sub(pc.start[p])
}

func (c *Camera) UpdatePtz(ctx context.Context, request *ptzpb.UpdatePtzRequest) (*ptzpb.Ptz, error) {
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

func (c *Camera) Stop(ctx context.Context, _ *ptzpb.StopPtzRequest) (*ptzpb.Ptz, error) {
	wg, ctx := errgroup.WithContext(ctx)

	// we don't know which command(s) are running, so stop them all!

	mu := sync.Mutex{}
	var multiErr error
	for _, command := range api.Commands {
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

func (c *Camera) PullMessages(_ *mqttpb.PullMessagesRequest, server mqttpb.MqttService_PullMessagesServer) error {
	changes := c.bus.Listen(server.Context())

	for change := range changes {
		asJson, err := json.Marshal(change)
		if err != nil {
			c.logger.Warn("unable to marshal message as JSON", zap.Error(err), zap.Any("change", change))
			continue
		}
		msg := &mqttpb.PullMessagesResponse{
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

func (c *Camera) PullControlTopics(request *udmipb.PullControlTopicsRequest, server udmipb.UdmiService_PullControlTopicsServer) error {
	return c.UnimplementedUdmiServiceServer.PullControlTopics(request, server)
}

func (c *Camera) OnMessage(ctx context.Context, request *udmipb.OnMessageRequest) (*udmipb.OnMessageResponse, error) {
	return c.UnimplementedUdmiServiceServer.OnMessage(ctx, request)
}

// GetExportMessage returns the camera's state, leaving out any point whose poll
// hasn't succeeded recently, or Unavailable if that leaves nothing.
//
// Unlike the drivers that read their device here, this returns what the polls last
// read: the state is assembled from several independent polls, and reading through
// them would also publish the result on PullExportMessages.
func (c *Camera) GetExportMessage(_ context.Context, _ *udmipb.GetExportMessageRequest) (*udmipb.MqttMessage, error) {
	c.lock.Lock()
	readAt, failed, readLimit := c.readAt, c.failed, c.readLimit
	state := c.copyState()
	c.lock.Unlock()

	now := c.now()
	stale := make(map[string]bool)
	var lastRead time.Time
	for p, fields := range pollFields {
		limit := max(c.staleAfter[p], 2*c.cycles.length(poll(p), now))
		if failed[p] {
			// the walk that failed on this camera may be slow for that very reason, and
			// allowing for it would bring the point back once that walk finished. Allow for
			// walks as they were when the point was read instead, which still tolerates one
			// failed poll.
			limit = readLimit[p]
		}
		if readAt[p].IsZero() || now.Sub(readAt[p]) > limit {
			for _, f := range fields {
				stale[f] = true
			}
		}
		if readAt[p].After(lastRead) {
			lastRead = readAt[p]
		}
	}
	points := udmiPoints(state, stale)
	if len(points) == 0 {
		if lastRead.IsZero() {
			return nil, status.Error(codes.Unavailable, "camera not read yet")
		}
		return nil, status.Errorf(codes.Unavailable, "camera last read %s ago", now.Sub(lastRead).Round(time.Second))
	}
	asJson, err := json.Marshal(points)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encode UDMI message")
	}
	return &udmipb.MqttMessage{Topic: c.udmiTopic(), Payload: string(asJson)}, nil
}

func (c *Camera) PullExportMessages(_ *udmipb.PullExportMessagesRequest, server udmipb.UdmiService_PullExportMessagesServer) error {
	changes := c.bus.Listen(server.Context())

	for change := range changes {
		asJson, err := marshalUDMIPayload(change)
		if err != nil {
			c.logger.Warn("unable to marshal message as JSON", zap.Error(err), zap.Any("change", change))
			continue
		}
		msg := &udmipb.PullExportMessagesResponse{
			Name: c.conf.Name,
			Message: &udmipb.MqttMessage{
				Topic:   c.udmiTopic(),
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

func (c *Camera) udmiTopic() string {
	return c.conf.Topic + "/event/pointset/points"
}

type udmiPoint struct {
	PresentValue any `json:"present_value"`
}

func marshalUDMIPayload(msg any) ([]byte, error) {
	return json.Marshal(udmiPoints(msg, nil))
}

// udmiPoints returns the fields of msg, a struct, as UDMI points, leaving out the
// fields named in skip.
func udmiPoints(msg any, skip map[string]bool) map[string]udmiPoint {
	out := make(map[string]udmiPoint)
	mt := reflect.TypeOf(msg)
	if mt.Kind() == reflect.Pointer {
		mt = mt.Elem()
	}
	mv := reflect.ValueOf(msg)
	if mv.Kind() == reflect.Pointer {
		mv = mv.Elem()
	}
	for i := 0; i < mt.NumField(); i++ {
		field := mt.Field(i)
		if skip[field.Name] {
			continue
		}
		// named and skipped by the json tag the way encoding/json does, so the points match
		// the camera's MQTT payload (SCB-1463)
		key := field.Name
		var omitEmpty bool
		if jsonTag := field.Tag.Get("json"); jsonTag == "-" {
			continue
		} else if jsonTag != "" {
			name, opts, _ := strings.Cut(jsonTag, ",")
			if name != "" {
				key = name
			}
			omitEmpty = slices.Contains(strings.Split(opts, ","), "omitempty")
		}

		value := mv.Field(i)
		if value.IsZero() && omitEmpty {
			continue
		}
		out[key] = udmiPoint{PresentValue: value.Interface()}
	}
	return out
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
			c.pollFailed(eventsPoll)
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
			c.pollFailed(occupancyPoll)
			break
		} else {
			if len(res.List) == 0 {
				logger.Warn("no people count data in response", zap.Any("res", res))
			} else if res.List[0].CameraIndexCode != c.conf.IndexCode {
				logger.Warn("response for unexpected camera", zap.String("got", res.List[0].CameraIndexCode), zap.String("want", c.conf.IndexCode))
				c.pollFailed(occupancyPoll)
				break
			} else {
				i := res.List[0]
				count := max(i.EnterNum-i.ExitNum, 0)
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
		c.pollFailed(streamPoll)
	} else {
		bytes, err := json.Marshal(res)
		if err != nil {
			logger.Warn("error serialising stream info", zap.Error(err))
			c.pollFailed(streamPoll)
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
		c.pollFailed(infoPoll)
	} else {
		active := res.Status == api.CameraStatusOnline
		c.updateActive(ctx, active)
	}
}

func (c *Camera) updateCount(ctx context.Context, count string) {
	c.updateAndNotify(ctx, occupancyPoll, func() {
		c.state.CamOcc = count
	})
}

func (c *Camera) updateVideo(ctx context.Context, video string) {
	c.updateAndNotify(ctx, streamPoll, func() {
		c.state.CamVideo = video
	})
}

func (c *Camera) updateFault(ctx context.Context, fault bool) {
	c.updateAndNotify(ctx, eventsPoll, func() {
		c.state.CamFlt = fault
	})
}

func (c *Camera) updateActive(ctx context.Context, active bool) {
	c.updateAndNotify(ctx, infoPoll, func() {
		c.state.CamState = active
	})
}

// updateAndNotify safely updates the camera state, following a successful read by p,
// and notifies listeners.
// The updateFn is called while holding the lock, then a copy is made and sent to the bus.
func (c *Camera) updateAndNotify(ctx context.Context, p poll, updateFn func()) {
	c.lock.Lock()
	updateFn()
	now := c.now()
	c.readAt[p] = now
	c.failed[p] = false
	c.readLimit[p] = max(c.staleAfter[p], 2*c.cycles.length(p, now))
	stateCopy := c.copyState()
	c.lock.Unlock()
	c.bus.Send(ctx, stateCopy)
}

// pollFailed records that p failed to read this camera.
func (c *Camera) pollFailed(p poll) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.failed[p] = true
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
