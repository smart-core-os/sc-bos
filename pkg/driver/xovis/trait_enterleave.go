package xovis

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
)

type enterLeaveServer struct {
	enterleavesensorpb.UnimplementedEnterLeaveSensorApiServer
	client      *client
	logicID     int
	multiSensor bool
	bus         *minibus.Bus[PushData]

	faultCheck *healthpb.FaultCheck
	pollInit   sync.Once
	poll       *task.Intermittent
	polls      *minibus.Bus[LiveLogicResponse]

	EnterLeaveTotal *resource.Value
}

func (e *enterLeaveServer) GetEnterLeaveEvent(ctx context.Context, request *enterleavesensorpb.GetEnterLeaveEventRequest) (*enterleavesensorpb.EnterLeaveEvent, error) {
	return e.read(ctx)
}

// read fetches the enter and leave totals from the sensor and records them in EnterLeaveTotal.
func (e *enterLeaveServer) read(ctx context.Context) (*enterleavesensorpb.EnterLeaveEvent, error) {
	res, err := getLiveLogic(ctx, e.client, e.multiSensor, e.logicID, e.faultCheck)
	if err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	totals := decodeEnterLeaveTotals(res.Logic.Counts)
	if totals == nil {
		return nil, errNotInOutLogic
	}

	_, _ = e.EnterLeaveTotal.Set(totals)
	return totals, nil
}

// watch polls the sensor, keeping EnterLeaveTotal current, until ctx is done.
func (e *enterLeaveServer) watch(ctx context.Context) {
	e.doPollInit()
	_ = e.poll.Attach(ctx) // can't error
}

func (e *enterLeaveServer) ResetEnterLeaveTotals(ctx context.Context, request *enterleavesensorpb.ResetEnterLeaveTotalsRequest) (*enterleavesensorpb.ResetEnterLeaveTotalsResponse, error) {
	return nil, resetLiveLogic(ctx, e.client, e.multiSensor, e.logicID, e.faultCheck)
}

func (e *enterLeaveServer) PullEnterLeaveEvents(request *enterleavesensorpb.PullEnterLeaveEventsRequest, server enterleavesensorpb.EnterLeaveSensorApi_PullEnterLeaveEventsServer) error {
	// get the initial value of the logics so we can compare later
	res, err := getLiveLogic(server.Context(), e.client, e.multiSensor, e.logicID, e.faultCheck)
	if err != nil {
		return status.Error(codes.Unavailable, err.Error())
	}

	totals := decodeEnterLeaveTotals(res.Logic.Counts)
	if totals == nil {
		return errNotInOutLogic
	}
	// push data identifies counts by id rather than name
	fwID, _, _ := findCountByName(res.Logic.Counts, "fw")
	bwID, _, _ := findCountByName(res.Logic.Counts, "bw")

	var lastSent *enterleavesensorpb.EnterLeaveEvent
	if !request.UpdatesOnly {
		elEvent := totals
		err := server.Send(&enterleavesensorpb.PullEnterLeaveEventsResponse{Changes: []*enterleavesensorpb.PullEnterLeaveEventsResponse_Change{
			{
				Name:            request.Name,
				ChangeTime:      timestamppb.Now(),
				EnterLeaveEvent: elEvent,
			},
		}})
		if err != nil {
			return err
		}
		lastSent = elEvent
	}

	// note: the accumulator continues to count totals even if the sensor is reset, for as long as the stream is active.
	accumulator := countAccumulator{
		forwardCountID:     fwID,
		backwardCountID:    bwID,
		forwardCountValue:  int(totals.GetEnterTotal()),
		backwardCountValue: int(totals.GetLeaveTotal()),
	}
	ctx := server.Context()
	e.doPollInit()
	polls := e.polls.Listen(ctx)
	webhooks := e.bus.Listen(ctx)

	// tell the polling logic we're interested
	_ = e.poll.Attach(ctx) // can't error

	eq := cmp.Equal()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-webhooks:
			if !ok {
				return nil
			}
			if data.LogicsData == nil {
				continue
			}
			records, ok := findLogicRecords(data.LogicsData, e.logicID)
			if !ok {
				continue
			}

			// note: accumulator totals are updated during consumeRecords. We want the values before this happens
			enterTotal, leaveTotal := int32(accumulator.forwardCountValue), int32(accumulator.backwardCountValue)
			events, err := accumulator.consumeRecords(records...)
			if err != nil {
				return err
			}

			if len(events) == 0 {
				continue
			}

			var enterLeaveChanges []*enterleavesensorpb.PullEnterLeaveEventsResponse_Change
			for _, event := range events {
				switch event.direction {
				case enterleavesensorpb.EnterLeaveEvent_ENTER:
					enterTotal++
				case enterleavesensorpb.EnterLeaveEvent_LEAVE:
					leaveTotal++
				}
				enterLeaveChanges = append(enterLeaveChanges, &enterleavesensorpb.PullEnterLeaveEventsResponse_Change{
					Name:       request.Name,
					ChangeTime: timestamppb.New(event.time),
					EnterLeaveEvent: &enterleavesensorpb.EnterLeaveEvent{
						Direction:  event.direction,
						EnterTotal: new(enterTotal),
						LeaveTotal: new(leaveTotal),
					},
				})
			}

			err = server.Send(&enterleavesensorpb.PullEnterLeaveEventsResponse{
				Changes: enterLeaveChanges,
			})
			if err != nil {
				return err
			}
			lastSent = enterLeaveChanges[len(enterLeaveChanges)-1].EnterLeaveEvent
		case data, ok := <-polls:
			if !ok {
				return nil
			}
			direction := lastSent.GetDirection()
			enterTotal, leaveTotal := accumulator.forwardCountValue, accumulator.backwardCountValue
			var reset bool
			if c, ok := findCountValueByID(data.Logic.Counts, fwID); ok {
				if c > accumulator.forwardCountValue {
					direction = enterleavesensorpb.EnterLeaveEvent_ENTER
				}
				if c < accumulator.forwardCountValue {
					reset = true
				}
				enterTotal = c
				accumulator.forwardCountValue = c
			}
			if c, ok := findCountValueByID(data.Logic.Counts, bwID); ok {
				if c > accumulator.backwardCountValue {
					direction = enterleavesensorpb.EnterLeaveEvent_LEAVE
				}
				if c < accumulator.backwardCountValue {
					reset = true
				}
				leaveTotal = c
				accumulator.backwardCountValue = c
			}
			if reset {
				// if any count decreased, we make no assumptions about direction
				direction = enterleavesensorpb.EnterLeaveEvent_DIRECTION_UNSPECIFIED
			}
			el := &enterleavesensorpb.EnterLeaveEvent{
				Direction:  direction,
				EnterTotal: new(int32(enterTotal)),
				LeaveTotal: new(int32(leaveTotal)),
			}
			if eq(lastSent, el) {
				continue
			}
			err = server.Send(&enterleavesensorpb.PullEnterLeaveEventsResponse{
				Changes: []*enterleavesensorpb.PullEnterLeaveEventsResponse_Change{
					{
						Name:            request.Name,
						ChangeTime:      timestamppb.New(data.Time),
						EnterLeaveEvent: el,
					},
				},
			})
			if err != nil {
				return err
			}
			lastSent = el
		}
	}
}

func (e *enterLeaveServer) doPollInit() {
	e.pollInit.Do(func() {
		e.polls = &minibus.Bus[LiveLogicResponse]{}
		e.poll = task.Poll(func(ctx context.Context) {
			res, err := getLiveLogic(ctx, e.client, e.multiSensor, e.logicID, e.faultCheck)
			if err != nil {
				// todo: log error
				return
			}
			if totals := decodeEnterLeaveTotals(res.Logic.Counts); totals != nil {
				_, _ = e.EnterLeaveTotal.Set(totals)
			}
			e.polls.Send(ctx, res)
		}, 30*time.Second)
	})
}

var errNotInOutLogic = status.Error(codes.FailedPrecondition,
	"Counts don't match expected structure; check that this is an InOut logic")

// decodeEnterLeaveTotals returns the totals of an InOut logic's counts, or nil if
// the counts don't have the fw and bw counts an InOut logic has.
func decodeEnterLeaveTotals(counts []Count) *enterleavesensorpb.EnterLeaveEvent {
	_, forwardCount, fwOK := findCountByName(counts, "fw")
	_, backwardCount, bwOK := findCountByName(counts, "bw")
	if !fwOK || !bwOK {
		return nil
	}
	return &enterleavesensorpb.EnterLeaveEvent{
		EnterTotal: new(int32(forwardCount)),
		LeaveTotal: new(int32(backwardCount)),
	}
}

func findLogicRecords(data *LogicsPushData, logicID int) (records []LogicRecord, ok bool) {
	for _, logic := range data.Logics {
		if logic.ID == logicID {
			records = logic.Records
			ok = true
			return
		}
	}
	return
}
