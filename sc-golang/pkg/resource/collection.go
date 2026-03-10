package resource

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/types"

	"github.com/smart-core-os/sc-golang/internal/minibus"
)

type Collection struct {
	*config

	mu   sync.RWMutex // protects byId and rng from concurrent access
	byId map[string]*item
	// "change" events contain a *CollectionChange instance
	bus minibus.Bus
}

func NewCollection(options ...Option) *Collection {
	conf := computeConfig(options...)
	initialItems := make(map[string]*item)
	for k, v := range conf.initialRecords {
		initialItems[k] = &item{body: v, changeTime: conf.clock.Now()}
	}
	conf.initialRecords = nil // so the gc can collect them

	c := &Collection{
		config: conf,
		byId:   initialItems,
		mu:     sync.RWMutex{},
	}

	return c
}

// Get will find the entry with the given ID. If no such entry exists, returns false.
func (c *Collection) Get(id string, opts ...ReadOption) (proto.Message, bool) {
	if c.idInterceptor != nil {
		id = c.idInterceptor(id)
	}

	readConfig := ComputeReadConfig(opts...)

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.byId[id]
	if !ok {
		return nil, false
	}

	return readConfig.FilterClone(entry.body), true
}

// List returns a list of all the entries, sorted by their ID.
func (c *Collection) List(opts ...ReadOption) []proto.Message {
	readConfig := ComputeReadConfig(opts...)

	c.mu.RLock()
	defer c.mu.RUnlock()
	tmp := c.itemSlice(readConfig)
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i].id < tmp[j].id
	})

	result := make([]proto.Message, 0, len(tmp))
	filter := readConfig.ResponseFilter()
	for _, e := range tmp {
		result = append(result, filter.FilterClone(e.body))
	}
	return result
}

// Add associates the given body with the id.
// If id already exists then an error is returned.
// A common set of options is WithGenIDIfAbsent() and WithIDCallback, which allows this function to generate a unique
// id if the given id is empty.
//
// Calling Add is equivalent to calling Update(id, body, WithExpectAbsent(), WithCreateIfAbsent(), opts...)
func (c *Collection) Add(id string, body proto.Message, opts ...WriteOption) (proto.Message, error) {
	opts = append([]WriteOption{
		WithExpectAbsent(), WithCreateIfAbsent(),
	}, opts...)
	return c.Update(id, body, opts...)
}

func (c *Collection) Update(id string, msg proto.Message, opts ...WriteOption) (proto.Message, error) {
	if c.idInterceptor != nil {
		id = c.idInterceptor(id)
	}

	writeRequest := ComputeWriteConfig(opts...)
	writer := writeRequest.fieldUpdater(c.writableFields)
	if err := writer.Validate(msg); err != nil {
		return nil, err
	}

	var created proto.Message // during create, this is returned by GetFn so concurrent reference checks pass
	oldValue, newValue, err := GetAndUpdate(
		&c.mu,
		func() (item proto.Message, err error) {
			if created != nil {
				return created, nil
			}

			// handle empty ids, generating them, and invoking callbacks
			if id == "" && writeRequest.genEmptyID {
				id, err = c.genID()
				if err != nil {
					return nil, err
				}
				if writeRequest.idCallback != nil {
					writeRequest.idCallback(id)
				}
			}

			val, exists := c.byId[id]
			if exists {
				if writeRequest.expectAbsent {
					return nil, ExpectAbsentPreconditionFailed
				}
				return val.body, nil
			}
			if !writeRequest.createIfAbsent {
				return nil, status.Errorf(codes.NotFound, "id %v not found", id)
			}
			created = msg.ProtoReflect().New().Interface()
			if writeRequest.createdCallback != nil {
				writeRequest.createdCallback()
			}
			return created, nil
		},
		writeRequest.changeFn(writer, msg),
		func(msg proto.Message) {
			c.byId[id] = &item{body: msg, changeTime: writeRequest.updateTime(c.clock)}
		})

	if err != nil {
		if s, ok := status.FromError(err); ok {
			return nil, status.Errorf(s.Code(), "%v %v", s.Message(), id)
		}
		return nil, err
	}
	changeType := types.ChangeType_UPDATE
	if oldValue == nil || created != nil {
		changeType = types.ChangeType_ADD
		oldValue = nil
	}
	c.bus.Send(context.TODO(), &CollectionChange{
		Id:         id,
		ChangeTime: writeRequest.updateTime(c.clock),
		ChangeType: changeType,
		OldValue:   oldValue,
		NewValue:   newValue,
	})
	return newValue, nil
}

// Delete removes the item with the given id from this collection.
// The removed item will be returned.
// If the id is unknown an error will be returned, unless WithAllowMissing is specified.
// Read-update-write operations can be checked via WithExpectedValue or WithExpectedCheck options.
func (c *Collection) Delete(id string, opts ...WriteOption) (proto.Message, error) {
	if c.idInterceptor != nil {
		id = c.idInterceptor(id)
	}

	args := ComputeWriteConfig(opts...)
	// Read lock first, we don't want to hold the lock when we pass control to callback functions
	c.mu.RLock()
	oldVal, exists := c.byId[id]
	c.mu.RUnlock()

	for attempt := 0; attempt < 5; attempt++ {
		if !exists {
			if !args.allowMissing {
				return nil, status.Error(codes.NotFound, "not found")
			}
			return nil, nil
		}
		if args.expectedCheck != nil {
			if err := args.expectedCheck(oldVal.body); err != nil {
				return oldVal.body, err
			}
		}
		if args.expectedValue != nil && !proto.Equal(oldVal.body, args.expectedValue) {
			return oldVal.body, ExpectedValuePreconditionFailed
		}

		c.mu.Lock()
		oldVal2, exists2 := c.byId[id]
		if oldVal2 != oldVal || exists2 != exists {
			// someone changed something while we were checking the value, try again
			c.mu.Unlock()
			oldVal, exists = oldVal2, exists2
			continue
		}

		// actually do the delete
		delete(c.byId, id)
		c.bus.Send(context.TODO(), &CollectionChange{
			Id:         id,
			ChangeTime: c.clock.Now(),
			ChangeType: types.ChangeType_REMOVE,
			OldValue:   oldVal.body,
		})
		c.mu.Unlock()
		return oldVal.body, nil
	}

	return nil, status.Error(codes.Unavailable, "concurrent writes")
}

func (c *Collection) Pull(ctx context.Context, opts ...ReadOption) <-chan *CollectionChange {
	readConfig := ComputeReadConfig(opts...)
	filter := readConfig.ResponseFilter()

	emit, currentValues := c.onUpdate(ctx, readConfig)
	send := make(chan *CollectionChange)

	go func() {
		defer close(send)

		if len(currentValues) > 0 {
			sort.Slice(currentValues, func(i, j int) bool {
				return currentValues[i].id < currentValues[j].id
			})
			lastIndex := len(currentValues) - 1
			for i, value := range currentValues {
				change := &CollectionChange{
					Id:            value.id,
					ChangeTime:    value.changeTime,
					ChangeType:    types.ChangeType_ADD,
					NewValue:      value.body,
					SeedValue:     true,
					LastSeedValue: i == lastIndex,
				}
				change = change.filter(filter)
				select {
				case <-ctx.Done():
					return
				case send <- change:
				}
			}
		}

		for event := range emit {
			change := event.(*CollectionChange)
			change, ok := change.include(readConfig.Include)
			if !ok {
				continue
			}
			change = change.filter(filter)
			if c.equivalence != nil && c.equivalence.Compare(change.OldValue, change.NewValue) {
				continue
			}
			select {
			case send <- change:
			case <-ctx.Done():
				return
			}
		}
	}()

	return send
}

// PullID subscribes to changes for a single item in the collection.
// The returned channel will close if ctx is Done or the item identified by id is deleted.
func (c *Collection) PullID(ctx context.Context, id string, opts ...ReadOption) <-chan *ValueChange {
	if c.idInterceptor != nil {
		id = c.idInterceptor(id)
	}

	send := make(chan *ValueChange)
	go func() {
		defer close(send)
		for change := range c.Pull(ctx, opts...) {
			if change.Id != id {
				continue
			}

			if change.ChangeType == types.ChangeType_REMOVE {
				return
			}

			// not sure how this case could happen, but let's deal with it anyway
			if change.NewValue == nil {
				log.Printf("WARN: CollectionChange.NewValue is nil, but not a REMOVE change! %v", change)
				return
			}

			select {
			case <-ctx.Done():
				return
			case send <- &ValueChange{ChangeTime: change.ChangeTime, Value: change.NewValue, SeedValue: change.SeedValue, LastSeedValue: change.LastSeedValue}:
			}
		}
	}()
	return send
}

func (c *Collection) onUpdate(ctx context.Context, config *ReadRequest) (<-chan any, []idItem) {
	var res []idItem
	if !config.UpdatesOnly {
		c.mu.RLock()
		defer c.mu.RUnlock()
		res = c.itemSlice(config)
	}

	ch := c.bus.Listen(ctx)
	if !config.Backpressure {
		ch = mergeCollectionExcess(ch)
	}

	return ch, res
}

// Clock returns the clock used by this resource for reporting time.
func (c *Collection) Clock() Clock {
	return c.clock
}

// itemSlice returns all the values in byId adjusted to match readConfig settings like ReadRequest.Include.
func (c *Collection) itemSlice(readConfig *ReadRequest) []idItem {
	res := make([]idItem, 0, len(c.byId))
	for id, value := range c.byId {
		if readConfig.Exclude(id, value.body) {
			continue
		}
		res = append(res, idItem{item: *value, id: id})
	}
	return res
}

func (c *Collection) genID() (string, error) {
	return GenerateUniqueId(c.rng, func(candidate string) bool {
		if c.idInterceptor != nil {
			candidate = c.idInterceptor(candidate)
		}
		_, exists := c.byId[candidate]
		return exists
	})
}

type item struct {
	body       proto.Message
	changeTime time.Time
}

type idItem struct {
	item
	id string
}
