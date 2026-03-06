package parentpb

import (
	"context"
	"fmt"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

// Model models a collection of children.
type Model struct {
	children *resource.Collection
}

func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		children: resource.NewCollection(args.childrenOpts...),
	}
}

// AddChild inserts a child into this model.
// If a child with the given Child.Name already exists, no changes will be made.
// Panics if child.Traits are not sorted in ascending order, required by AddChildTrait.
func (m *Model) AddChild(child *traits.Child) {
	if err := validateChild(child); err != nil {
		panic(err)
	}
	_, _ = m.children.Add(child.Name, child)
}

func validateChild(child *traits.Child) error {
	if child.Name == "" {
		return fmt.Errorf("child has no name")
	}
	sorted := sort.SliceIsSorted(child.Traits, func(i, j int) bool {
		return child.Traits[i].Name < child.Traits[j].Name
	})
	if !sorted {
		return fmt.Errorf("%v Traits are not sorted: %v", child.Name, child.Traits)
	}

	return nil
}

// RemoveChildByName removes a child from this model matching the given name.
func (m *Model) RemoveChildByName(name string, opts ...resource.WriteOption) (*traits.Child, error) {
	msg, err := m.children.Delete(name, opts...)
	if msg == nil {
		return nil, err
	}
	return msg.(*traits.Child), err
}

// AddChildTrait ensures that a child with the given name and list of trait names exists in this model.
// If no child with the given name is already know, one will be created.
// If a child is already known with the given name, its traits will be unioned with the given trait names.
func (m *Model) AddChildTrait(name string, traitName ...trait.Name) (child *traits.Child, created bool) {
	msg, err := m.children.Update(name, &traits.Child{Name: name},
		resource.WithCreateIfAbsent(),
		resource.WithCreatedCallback(func() {
			created = true
		}),
		resource.InterceptBefore(func(old, value proto.Message) {
			oldChild := old.(*traits.Child)
			newChild := value.(*traits.Child)
			newChild.Traits = traitUnion(oldChild.Traits, traitName...)
		}))
	if err != nil {
		panic(err) // shouldn't happen
	}
	return msg.(*traits.Child), created
}

// RemoveChildTrait ensures that the named child no longer mentions they support the given trait names.
// If no child exists with the given name then nil will be returned.
func (m *Model) RemoveChildTrait(name string, traitName ...trait.Name) *traits.Child {
	msg, err := m.children.Update(name, &traits.Child{Name: name},
		resource.InterceptBefore(func(old, value proto.Message) {
			oldChild := old.(*traits.Child)
			newChild := value.(*traits.Child)
			newChild.Traits = traitRemove(oldChild.Traits, traitName...)
		}))
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			return nil
		}
		panic(err) // NotFound is the only error we expect
	}
	return msg.(*traits.Child)
}

// ListChildren returns a slice of all known Child instances.
func (m *Model) ListChildren() []*traits.Child {
	msgs := m.children.List()
	children := make([]*traits.Child, len(msgs))
	for i, msg := range msgs {
		children[i] = msg.(*traits.Child)
	}
	return children
}

// PullChildren returns a chan that will emit when changes are made to the known children of this model.
func (m *Model) PullChildren(ctx context.Context, opts ...resource.ReadOption) <-chan *traits.PullChildrenResponse_Change {
	out := make(chan *traits.PullChildrenResponse_Change)
	changes := m.children.Pull(ctx, opts...)

	go func() {
		defer close(out)
		for change := range changes {
			out <- childrenChangeToProto(change)
		}
	}()

	return out
}

func childrenChangeToProto(change *resource.CollectionChange) *traits.PullChildrenResponse_Change {
	pChange := traits.PullChildrenResponse_Change{
		Type:       change.ChangeType,
		ChangeTime: timestamppb.New(change.ChangeTime),
	}
	if change.OldValue != nil {
		pChange.OldValue = change.OldValue.(*traits.Child)
	}
	if change.NewValue != nil {
		pChange.NewValue = change.NewValue.(*traits.Child)
	}
	return &pChange
}

// traitUnion returns a slice containing the union of has and more (converted to *traits.Trait).
// The has slice should be sorted in ascending order by Trait.Name.
// The returned slice will be sorted in ascending order by Trait.Name.
func traitUnion(has []*traits.Trait, more ...trait.Name) []*traits.Trait {
	// has should be sorted by Trait.Name
	for _, t := range more {
		ts := string(t)
		insertIndex := sort.Search(len(has), func(i int) bool {
			return has[i].Name >= ts
		})
		switch {
		case insertIndex == len(has): // append to end
			has = append(has, &traits.Trait{Name: ts})
		case has[insertIndex].Name == ts: // already exists, do nothing
		default: // insert into slice
			has = append(has[:insertIndex+1], has[insertIndex:]...)
			has[insertIndex] = &traits.Trait{Name: ts}
		}
	}
	return has
}

// traitRemove returns a slice containing only traits from has that aren't in remove.
// The has slice should be sorted in ascending order by Trait.Name.
// The returned slice will be sorted in ascending order by Trait.Name.
func traitRemove(has []*traits.Trait, remove ...trait.Name) []*traits.Trait {
	// has should be sorted by Trait.Name
	for _, t := range remove {
		ts := string(t)
		insertIndex := sort.Search(len(has), func(i int) bool {
			return has[i].Name >= ts
		})
		if insertIndex == len(has) {
			continue // t isn't in has, nothing to do this iteration
		}
		copy(has[insertIndex:], has[insertIndex+1:])
		has = has[:len(has)-1]
	}
	return has
}
