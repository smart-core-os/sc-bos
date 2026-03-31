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
	res, err := getLiveLogic(ctx, e.client, e.multiSensor, e.logicID, e.faultCheck)
	if err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	_, forwardCount, fwOK := findCountByName(res.Logic.Counts, "fw")
	_, backwardCount, bwOK := findCountByName(res.Logic.Counts, "bw")
	if !fwOK || !bwOK {
		return nil, status.Error(codes.FailedPrecondition,
			"Counts don't match expected structure; check that this is an InOut logic")
	}

	forwardCount32, backwardCount32 := int32(forwardCount), int32(backwardCount)

	e.EnterLeaveTotal.Set(&enterleavesensorpb.EnterLeaveEvent{
		EnterTotal: &forwardCount32,
		LeaveTotal: &backwardCount32,
	})

	return &enterleavesensorpb.EnterLeaveEvent{
		EnterTotal: &forwardCount32,
		LeaveTotal: &backwardCount32,
	}, nil
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

	fwID, forwardCount, fwOK := findCountByName(res.Logic.Counts, "fw")
	bwID, backwardCount, bwOK := findCountByName(res.Logic.Counts, "bw")
	if !fwOK || !bwOK {
		return status.Error(codes.FailedPrecondition,
			"Counts don't match expected structure; check that this is an InOut logic")
	}

	var lastSent *enterleavesensorpb.EnterLeaveEvent
	if !request.UpdatesOnly {
		enterTotal, leaveTotal := int32(forwardCount), int32(backwardCount)
		elEvent := &enterleavesensorpb.EnterLeaveEvent{
			EnterTotal: &enterTotal,
			LeaveTotal: &leaveTotal,
		}
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
		forwardCountValue:  forwardCount,
		backwardCountValue: backwardCount,
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
						EnterTotal: &enterTotal,
						LeaveTotal: &leaveTotal,
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
				if c < accumulator.forwardCountValue {
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
			e.polls.Send(ctx, res)
		}, 30*time.Second)
	})
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
