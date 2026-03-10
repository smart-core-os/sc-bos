package openclosepb

import (
	"context"
	"fmt"
	"slices"
	"time"

	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	positions *resource.Collection // of *traits.OpenClosePosition
	presets   []preset
}

// NewModel creates a new *Model with the given options.
// Options are applied to all resource types in the model unless specified otherwise in the option.
func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		positions: resource.NewCollection(args.positionsOpts...),
		presets:   args.presets,
	}
}

// preset describes a proto preset combined with the positions applied when activating that preset.
type preset struct {
	desc      *traits.OpenClosePositions_Preset
	positions []*traits.OpenClosePosition
}

func (m *Model) GetPositions(opts ...resource.ReadOption) (*traits.OpenClosePositions, error) {
	allPositions := m.positions.List(opts...) // already sorted by ID aka Direction ordinal
	dst := &traits.OpenClosePositions{
		States: make([]*traits.OpenClosePosition, len(allPositions)),
	}
	for i, position := range allPositions {
		dst.States[i] = position.(*traits.OpenClosePosition)
	}

	preset, _ := m.presetForValue(dst.States)
	if preset != nil {
		dst.Preset = preset
	}

	return dst, nil
}

func (m *Model) GetPosition(dir traits.OpenClosePosition_Direction, opts ...resource.ReadOption) (*traits.OpenClosePosition, error) {
	position, ok := m.positions.Get(directionToID(dir), opts...)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "position %v not found", dir)
	}
	return position.(*traits.OpenClosePosition), nil
}

func (m *Model) UpdatePositions(positions *traits.OpenClosePositions, opts ...resource.WriteOption) (*traits.OpenClosePositions, error) {
	// preset handling
	if positions.Preset != nil {
		preset, presetPositions := m.presetForName(positions.Preset.Name)
		if preset == nil {
			return nil, status.Errorf(codes.InvalidArgument, "preset %q not found", positions.Preset.Name)
		}
		positions.States = presetPositions
	}

	writeRequest := resource.ComputeWriteConfig(opts...)
	opts = append([]resource.WriteOption{}, opts...)
	if writeRequest.UpdateMask != nil {
		opts = append(opts, resource.WithUpdateMask(masks.RemovePrefix("states", writeRequest.UpdateMask)))
	}
	opts = append(opts, resource.WithCreateIfAbsent())

	for _, state := range positions.States {
		_, err := m.positions.Update(directionToID(state.Direction), state, opts...)
		if err != nil {
			return nil, err
		}
	}

	return m.GetPositions()
}

func (m *Model) UpdatePosition(position *traits.OpenClosePosition, opts ...resource.WriteOption) (*traits.OpenClosePosition, error) {
	return m.UpdatePositionN(position.Direction, position, opts...)
}

func (m *Model) UpdatePositionN(dir traits.OpenClosePosition_Direction, position *traits.OpenClosePosition, opts ...resource.WriteOption) (*traits.OpenClosePosition, error) {
	msg, err := m.positions.Update(directionToID(dir), position, opts...)
	if err != nil {
		return nil, err
	}
	return msg.(*traits.OpenClosePosition), nil
}

func (m *Model) PullPositions(ctx context.Context, ops ...resource.ReadOption) <-chan PullOpenClosePositionsChange {
	readRequest := resource.ComputeReadConfig(ops...)
	responseFilter := readRequest.ResponseFilter()
	eq := cmp.Equal()

	send := make(chan PullOpenClosePositionsChange)
	go func() {
		defer close(send)
		all := make(map[string]*traits.OpenClosePosition, 1)
		seenAll := false
		var last *traits.OpenClosePositions

		for change := range m.positions.Pull(ctx) {
			if change.NewValue == nil {
				delete(all, change.Id)
			} else {
				all[change.Id] = change.NewValue.(*traits.OpenClosePosition)
			}

			shouldSend := seenAll || (change.LastSeedValue && !readRequest.UpdatesOnly)
			if change.LastSeedValue {
				seenAll = true
			}
			if !shouldSend {
				continue
			}

			// transform into the correct output format
			positions := &traits.OpenClosePositions{
				States: maps.Values(all),
			}
			sortPositions(positions.States)

			positions.Preset, _ = m.presetForValue(positions.States)

			// projection and filtering
			responseFilter.Filter(positions)
			if eq(last, positions) {
				continue
			}
			last = positions

			// do the send
			select {
			case <-ctx.Done():
				return
			case send <- PullOpenClosePositionsChange{
				Positions:  positions,
				ChangeTime: change.ChangeTime,
			}:
			}
		}
	}()
	return send
}

type PullOpenClosePositionsChange struct {
	Positions  *traits.OpenClosePositions
	ChangeTime time.Time
}

func (m *Model) ListPresets() []*traits.OpenClosePositions_Preset {
	presets := make([]*traits.OpenClosePositions_Preset, len(m.presets))
	for i, preset := range m.presets {
		presets[i] = preset.desc
	}
	return presets
}

func (m *Model) HasPreset(name string) bool {
	n, _ := m.presetForName(name)
	return n != nil
}

func (m *Model) presetForName(name string) (*traits.OpenClosePositions_Preset, []*traits.OpenClosePosition) {
	for _, preset := range m.presets {
		if preset.desc.Name == name {
			return preset.desc, preset.positions
		}
	}
	return nil, nil
}

func (m *Model) presetForValue(value []*traits.OpenClosePosition) (*traits.OpenClosePositions_Preset, []*traits.OpenClosePosition) {
	for _, preset := range m.presets {
		if proto.Equal(&traits.OpenClosePositions{States: preset.positions}, &traits.OpenClosePositions{States: value}) {
			return preset.desc, preset.positions
		}
	}
	return nil, nil
}

func directionToID(direction traits.OpenClosePosition_Direction) string {
	// if the number of directions ever exceeds 99 we need to change this
	return fmt.Sprintf("%02d", direction)
}
func sortPositions(ps []*traits.OpenClosePosition) {
	slices.SortFunc(ps, func(a, b *traits.OpenClosePosition) int {
		return int(a.Direction - b.Direction)
	})
}
