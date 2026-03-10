package lightpb

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/group"
)

// Group combines multiple named devices into a single named device.
type Group struct {
	traits.UnimplementedLightApiServer

	ReadExecution  group.ExecutionStrategy
	WriteExecution group.ExecutionStrategy

	members []string
	impl    traits.LightApiClient
}

// NewGroup creates a new Group instance with ExecutionStrategyAll for both reads and writes.
func NewGroup(impl traits.LightApiClient, members ...string) *Group {
	return &Group{
		ReadExecution:  group.ExecutionStrategyAll,
		WriteExecution: group.ExecutionStrategyAll,
		impl:           impl,
		members:        members,
	}
}

func (s *Group) GetBrightness(ctx context.Context, request *traits.GetBrightnessRequest) (*traits.Brightness, error) {
	actions := make([]group.Member, len(s.members))
	for i, member := range s.members {
		i := i
		member := member
		actions[i] = func(ctx context.Context) (proto.Message, error) {
			memberRequest := proto.Clone(request).(*traits.GetBrightnessRequest)
			memberRequest.Name = member
			return s.impl.GetBrightness(ctx, memberRequest)
		}
	}
	results, err := group.Execute(ctx, s.ReadExecution, actions)
	if err != nil {
		return nil, err
	}

	return s.reduce(results), nil
}

func (s *Group) UpdateBrightness(ctx context.Context, request *traits.UpdateBrightnessRequest) (*traits.Brightness, error) {
	actions := make([]group.Member, len(s.members))
	for i, member := range s.members {
		i := i
		member := member
		actions[i] = func(ctx context.Context) (proto.Message, error) {
			memberRequest := proto.Clone(request).(*traits.UpdateBrightnessRequest)
			memberRequest.Name = member
			return s.impl.UpdateBrightness(ctx, memberRequest)
		}
	}
	results, err := group.Execute(ctx, s.WriteExecution, actions)
	if err != nil {
		return nil, err
	}

	return s.reduce(results), nil
}

func (s *Group) PullBrightness(request *traits.PullBrightnessRequest, server traits.LightApi_PullBrightnessServer) error {
	// NB we dont connect response headers or trailers for the members with the passed server.
	// If we did we'd be in a situation where one member who didn't send headers could cause
	// the entire subscription to be blocked. Either that or we'd be introducing timeouts and latency.
	memberValues := make(chan pullBrightnessResponse)

	actions := s.pullBrightnessActions(request, memberValues)

	ctx, cancelFunc := context.WithCancel(server.Context())
	defer cancelFunc() // just to be sure, it's likely that normal return will cancel the server context anyway

	returnErr := make(chan error, 1)
	go func() {
		_, err := group.Execute(ctx, s.ReadExecution, actions)
		returnErr <- err
	}()

	lastChange := new(traits.PullBrightnessResponse_Change)
	memberChanges := make([]*traits.PullBrightnessResponse_Change, len(s.members))

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
			newChange := s.reduceBrightnessChanges(memberChanges)
			if proto.Equal(lastChange, newChange) {
				continue
			}
			lastChange = newChange
			toSend := proto.Clone(lastChange).(*traits.PullBrightnessResponse_Change)
			toSend.Name = request.Name
			toSend.ChangeTime = endChange.ChangeTime
			if toSend.ChangeTime == nil {
				toSend.ChangeTime = timestamppb.Now()
			}
			err := server.Send(&traits.PullBrightnessResponse{
				Changes: []*traits.PullBrightnessResponse_Change{toSend},
			})
			if err != nil {
				cancelFunc()
				<-returnErr // wait for all the members to complete
				return err
			}
		}
	}
}

func (s *Group) pullBrightnessActions(request *traits.PullBrightnessRequest, memberValues chan<- pullBrightnessResponse) []group.Member {
	actions := make([]group.Member, len(s.members))
	for i, member := range s.members {
		i := i
		member := member
		actions[i] = func(ctx context.Context) (msg proto.Message, err error) {
			memberRequest := proto.Clone(request).(*traits.PullBrightnessRequest)
			memberRequest.Name = member
			stream, err := s.impl.PullBrightness(ctx, memberRequest)
			if err != nil {
				return
			}

			// NB ctx cancellation is handled by the Recv method
			for {
				// read a message
				var response *traits.PullBrightnessResponse
				response, err = stream.Recv()
				if err != nil {
					break
				}
				select {
				case memberValues <- pullBrightnessResponse{i, response}:
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

func (s *Group) reduce(results []proto.Message) *traits.Brightness {
	val := new(traits.Brightness)
	for i, result := range results {
		if result == nil {
			continue
		}
		typedResult := result.(*traits.Brightness)
		val = s.reduceBrightness(val, typedResult, i)
	}
	return val
}

func (s *Group) reduceBrightnessChanges(arr []*traits.PullBrightnessResponse_Change) *traits.PullBrightnessResponse_Change {
	val := &traits.PullBrightnessResponse_Change{}
	for i, change := range arr {
		if change == nil {
			// nil changes happen because the incoming array can be partially populated
			// depending on whether we've received anything from a group member
			continue
		}
		val.Brightness = s.reduceBrightness(val.Brightness, change.Brightness, i)
	}
	return val
}

func (s *Group) reduceBrightness(acc, v *traits.Brightness, i int) *traits.Brightness {
	if v == nil {
		return acc
	}
	if acc == nil {
		val := &traits.Brightness{}
		proto.Merge(val, v)
		return val
	}

	// average strategy
	acc.LevelPercent = (acc.LevelPercent*float32(i) + v.LevelPercent) / (float32(i) + 1)

	return acc
}

type pullBrightnessResponse struct {
	i int
	m *traits.PullBrightnessResponse
}
