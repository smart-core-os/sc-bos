package onoffpb

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/group"
)

// Group combines multiple named devices into a single named device.
type Group struct {
	UnimplementedOnOffApiServer

	ReadExecution  group.ExecutionStrategy
	WriteExecution group.ExecutionStrategy

	members []string
	impl    OnOffApiClient
}

// NewGroup creates a new Group instance with ExecutionStrategyAll for both reads and writes.
func NewGroup(impl OnOffApiClient, members ...string) *Group {
	return &Group{
		ReadExecution:  group.ExecutionStrategyAll,
		WriteExecution: group.ExecutionStrategyAll,
		impl:           impl,
		members:        members,
	}
}

func (s *Group) GetOnOff(ctx context.Context, request *GetOnOffRequest) (*OnOff, error) {
	actions := make([]group.Member, len(s.members))
	for i, member := range s.members {
		i := i
		member := member
		actions[i] = func(ctx context.Context) (proto.Message, error) {
			memberRequest := proto.Clone(request).(*GetOnOffRequest)
			memberRequest.Name = member
			return s.impl.GetOnOff(ctx, memberRequest)
		}
	}
	results, err := group.Execute(ctx, s.ReadExecution, actions)
	if err != nil {
		return nil, err
	}

	return s.reduce(results), nil
}

func (s *Group) UpdateOnOff(ctx context.Context, request *UpdateOnOffRequest) (*OnOff, error) {
	actions := make([]group.Member, len(s.members))
	for i, member := range s.members {
		i := i
		member := member
		actions[i] = func(ctx context.Context) (proto.Message, error) {
			memberRequest := proto.Clone(request).(*UpdateOnOffRequest)
			memberRequest.Name = member
			return s.impl.UpdateOnOff(ctx, memberRequest)
		}
	}
	results, err := group.Execute(ctx, s.WriteExecution, actions)
	if err != nil {
		return nil, err
	}

	return s.reduce(results), nil
}

func (s *Group) PullOnOff(request *PullOnOffRequest, server OnOffApi_PullOnOffServer) error {
	// NB we dont connect response headers or trailers for the members with the passed server.
	// If we did we'd be in a situation where one member who didn't send headers could cause
	// the entire subscription to be blocked. Either that or we'd be introducing timeouts and latency.
	memberValues := make(chan pullOnOffResponse)

	actions := s.pullOnOffActions(request, memberValues)

	ctx, cancelFunc := context.WithCancel(server.Context())
	defer cancelFunc() // just to be sure, it's likely that normal return will cancel the server context anyway

	returnErr := make(chan error, 1)
	go func() {
		_, err := group.Execute(ctx, s.ReadExecution, actions)
		returnErr <- err
	}()

	lastChange := new(PullOnOffResponse_Change)
	memberChanges := make([]*PullOnOffResponse_Change, len(s.members))

	for {
		select {
		// We shouldn't need to have a ctx.Done case as the member actions
		// all listen to this already and should return in that case eventually
		// causing returnErr to have a value
		case err := <-returnErr:
			return err
		case msg := <-memberValues:
			if len(msg.m.Changes) == 0 {
				continue
			}
			// todo: work out the list of changes to send not just this final change
			endChange := msg.m.Changes[len(msg.m.Changes)-1]
			memberChanges[msg.i] = endChange
			newChange := s.reduceOnOffChanges(memberChanges)
			if proto.Equal(lastChange, newChange) {
				continue
			}
			lastChange = newChange
			toSend := proto.Clone(lastChange).(*PullOnOffResponse_Change)
			toSend.Name = request.Name
			toSend.ChangeTime = endChange.ChangeTime
			if toSend.ChangeTime == nil {
				toSend.ChangeTime = timestamppb.Now()
			}
			err := server.Send(&PullOnOffResponse{
				Changes: []*PullOnOffResponse_Change{toSend},
			})
			if err != nil {
				cancelFunc()
				<-returnErr // wait for all the members to complete
				return err
			}
		}
	}
}

func (s *Group) pullOnOffActions(request *PullOnOffRequest, memberValues chan<- pullOnOffResponse) []group.Member {
	actions := make([]group.Member, len(s.members))
	for i, member := range s.members {
		i := i
		member := member
		actions[i] = func(ctx context.Context) (msg proto.Message, err error) {
			memberRequest := proto.Clone(request).(*PullOnOffRequest)
			memberRequest.Name = member
			stream, err := s.impl.PullOnOff(ctx, memberRequest)
			if err != nil {
				return
			}

			// NB ctx cancellation is handled by the Recv method
			for {
				// read a message
				var response *PullOnOffResponse
				response, err = stream.Recv()
				if err != nil {
					break
				}
				select {
				case memberValues <- pullOnOffResponse{i, response}:
				case <-ctx.Done():
					err = ctx.Err()
					return
				}
			}

			return
		}
	}
	return actions
}

func (s *Group) reduce(results []proto.Message) *OnOff {
	val := new(OnOff)
	for _, result := range results {
		if result == nil {
			continue
		}
		typedResult := result.(*OnOff)
		val = s.reduceOnOff(val, typedResult)
	}
	return val
}

func (s *Group) reduceOnOffChanges(arr []*PullOnOffResponse_Change) *PullOnOffResponse_Change {
	val := &PullOnOffResponse_Change{}
	for _, change := range arr {
		if change == nil {
			// nil changes happen because the incoming array can be partially populated
			// depending on whether we've received anything from a group member
			continue
		}
		val.OnOff = s.reduceOnOff(val.OnOff, change.OnOff)
	}
	return val
}

func (s *Group) reduceOnOff(acc, v *OnOff) *OnOff {
	if v == nil {
		return acc
	}
	if acc == nil {
		val := &OnOff{}
		proto.Merge(val, v)
		return val
	}

	// max strategy
	if acc.State == OnOff_STATE_UNSPECIFIED {
		acc.State = v.State
	} else if v.State == OnOff_ON {
		acc.State = OnOff_ON
	}

	return acc
}

type pullOnOffResponse struct {
	i int
	m *PullOnOffResponse
}
