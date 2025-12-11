package xovis

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type enterLeaveServer struct {
	traits.UnimplementedEnterLeaveSensorApiServer
	client      *client
	logicID     int
	multiSensor bool
	bus         *minibus.Bus[PushData]

	pollInit sync.Once
	poll     *task.Intermittent
	polls    *minibus.Bus[LiveLogicResponse]

	EnterLeaveTotal *resource.Value
}

func (e *enterLeaveServer) GetEnterLeaveEvent(ctx context.Context, request *traits.GetEnterLeaveEventRequest) (*traits.EnterLeaveEvent, error) {
	res, err := getLiveLogic(e.client, e.multiSensor, e.logicID)
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

	e.EnterLeaveTotal.Set(&traits.EnterLeaveEvent{
		EnterTotal: &forwardCount32,
		LeaveTotal: &backwardCount32,
	})

	return &traits.EnterLeaveEvent{
		EnterTotal: &forwardCount32,
		LeaveTotal: &backwardCount32,
	}, nil
}

func (e *enterLeaveServer) ResetEnterLeaveTotals(ctx context.Context, request *traits.ResetEnterLeaveTotalsRequest) (*traits.ResetEnterLeaveTotalsResponse, error) {
	return nil, resetLiveLogic(e.client, e.multiSensor, e.logicID)
}

func (e *enterLeaveServer) PullEnterLeaveEvents(request *traits.PullEnterLeaveEventsRequest, server traits.EnterLeaveSensorApi_PullEnterLeaveEventsServer) error {
	// get the initial value of the logics so we can compare later
	res, err := getLiveLogic(e.client, e.multiSensor, e.logicID)
	if err != nil {
		return status.Error(codes.Unavailable, err.Error())
	}

	fwID, forwardCount, fwOK := findCountByName(res.Logic.Counts, "fw")
	bwID, backwardCount, bwOK := findCountByName(res.Logic.Counts, "bw")
	if !fwOK || !bwOK {
		return status.Error(codes.FailedPrecondition,
			"Counts don't match expected structure; check that this is an InOut logic")
	}

	var lastSent *traits.EnterLeaveEvent
	if !request.UpdatesOnly {
		enterTotal, leaveTotal := int32(forwardCount), int32(backwardCount)
		elEvent := &traits.EnterLeaveEvent{
			EnterTotal: &enterTotal,
			LeaveTotal: &leaveTotal,
		}
		err := server.Send(&traits.PullEnterLeaveEventsResponse{Changes: []*traits.PullEnterLeaveEventsResponse_Change{
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

			var enterLeaveChanges []*traits.PullEnterLeaveEventsResponse_Change
			for _, event := range events {
				switch event.direction {
				case traits.EnterLeaveEvent_ENTER:
					enterTotal++
				case traits.EnterLeaveEvent_LEAVE:
					leaveTotal++
				}
				enterLeaveChanges = append(enterLeaveChanges, &traits.PullEnterLeaveEventsResponse_Change{
					Name:       request.Name,
					ChangeTime: timestamppb.New(event.time),
					EnterLeaveEvent: &traits.EnterLeaveEvent{
						Direction:  event.direction,
						EnterTotal: &enterTotal,
						LeaveTotal: &leaveTotal,
					},
				})
			}

			err = server.Send(&traits.PullEnterLeaveEventsResponse{
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
					direction = traits.EnterLeaveEvent_ENTER
				}
				if c < accumulator.forwardCountValue {
					reset = true
				}
				enterTotal = c
				accumulator.forwardCountValue = c
			}
			if c, ok := findCountValueByID(data.Logic.Counts, bwID); ok {
				if c > accumulator.backwardCountValue {
					direction = traits.EnterLeaveEvent_LEAVE
				}
				if c < accumulator.forwardCountValue {
					reset = true
				}
				leaveTotal = c
				accumulator.backwardCountValue = c
			}
			if reset {
				// if any count decreased, we make no assumptions about direction
				direction = traits.EnterLeaveEvent_DIRECTION_UNSPECIFIED
			}
			el := &traits.EnterLeaveEvent{
				Direction:  direction,
				EnterTotal: ptr(int32(enterTotal)),
				LeaveTotal: ptr(int32(leaveTotal)),
			}
			if eq(lastSent, el) {
				continue
			}
			err = server.Send(&traits.PullEnterLeaveEventsResponse{
				Changes: []*traits.PullEnterLeaveEventsResponse_Change{
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
			res, err := getLiveLogic(e.client, e.multiSensor, e.logicID)
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

func ptr[T any](v T) *T {
	return &v
}
