package vendingpb

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// Model describes the data structure needed to implement the Vending trait.
type Model struct {
	inventory   *resource.Collection // of *traits.Consumable_Stock, keyed by consumable name
	consumables *resource.Collection // of *traits.Consumable, keyed by consumable name
}

// NewModel creates a new Model with the given options.
// Options from the resource package are applied to all resources of this model.
// Use WithInventoryOption or WithConsumablesOption to target a specific resource.
// See WithInitialStock and WithInitialConsumable for simple ways to pre-populate this model.
func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		inventory:   resource.NewCollection(args.inventoryOptions...),
		consumables: resource.NewCollection(args.consumableOptions...),
	}
}

// CreateConsumable creates a new consumable record.
// If consumable.Name is specified it will be used as the key, it absent a new name will be invented.
// If the consumables name already exists, an error will be returned.
func (m *Model) CreateConsumable(consumable *Consumable) (*Consumable, error) {
	return castConsumable(m.consumables.Add(consumable.Name, consumable, resource.WithGenIDIfAbsent(), resource.WithIDCallback(func(id string) {
		consumable.Name = id
	})))
}

func (m *Model) GetConsumable(name string, opts ...resource.ReadOption) (*Consumable, bool) {
	msg, exists := m.consumables.Get(name, opts...)
	if msg == nil {
		return nil, exists
	}
	return msg.(*Consumable), exists
}

func (m *Model) UpdateConsumable(consumable *Consumable, opts ...resource.WriteOption) (*Consumable, error) {
	if consumable.Name == "" {
		return nil, status.Error(codes.NotFound, "name not specified")
	}
	msg, err := m.consumables.Update(consumable.Name, consumable, opts...)
	return castConsumable(msg, err)
}

func (m *Model) DeleteConsumable(name string, opts ...resource.WriteOption) (*Consumable, error) {
	return castConsumable(m.consumables.Delete(name, opts...))
}

type ConsumableChange struct {
	ChangeTime time.Time
	Value      *Consumable
}

func (m *Model) PullConsumable(ctx context.Context, name string, opts ...resource.ReadOption) <-chan ConsumableChange {
	send := make(chan ConsumableChange)
	go func() {
		defer close(send)
		for change := range m.consumables.PullID(ctx, name, opts...) {
			select {
			case <-ctx.Done():
				return
			case send <- ConsumableChange{ChangeTime: change.ChangeTime, Value: change.Value.(*Consumable)}:
			}
		}
	}()
	return send
}

func (m *Model) ListConsumables(opts ...resource.ReadOption) []*Consumable {
	msgs := m.consumables.List(opts...)
	res := make([]*Consumable, len(msgs))
	for i, msg := range msgs {
		res[i] = msg.(*Consumable)
	}
	return res
}

type ConsumablesChange struct {
	ID         string
	ChangeTime time.Time
	ChangeType typespb.ChangeType
	OldValue   *Consumable
	NewValue   *Consumable
}

func (m *Model) PullConsumables(ctx context.Context, opts ...resource.ReadOption) <-chan ConsumablesChange {
	send := make(chan ConsumablesChange)
	go func() {
		defer close(send)
		for change := range m.consumables.Pull(ctx, opts...) {
			oldVal, newVal := castConsumableChange(change)
			event := ConsumablesChange{
				ID:         change.Id,
				ChangeTime: change.ChangeTime,
				ChangeType: change.ChangeType,
				OldValue:   oldVal,
				NewValue:   newVal,
			}
			select {
			case <-ctx.Done():
				return
			case send <- event:
			}
		}
	}()
	return send
}

// CreateStock adds a stock record to this model.
// If stock.Consumable is not supplied, a new consumable name will be invented.
// Errors if the stock.Consumable already exists as a known stock entry.
func (m *Model) CreateStock(stock *Consumable_Stock) (*Consumable_Stock, error) {
	return castStock(m.inventory.Add(stock.Consumable, stock, resource.WithGenIDIfAbsent(), resource.WithIDCallback(func(id string) {
		stock.Consumable = id
	})))
}

func (m *Model) GetStock(consumable string, opts ...resource.ReadOption) (*Consumable_Stock, bool) {
	msg, exists := m.inventory.Get(consumable, opts...)
	if msg == nil {
		return nil, exists
	}
	return msg.(*Consumable_Stock), exists
}

func (m *Model) UpdateStock(stock *Consumable_Stock, opts ...resource.WriteOption) (*Consumable_Stock, error) {
	if stock.Consumable == "" {
		return nil, status.Error(codes.NotFound, "consumable not specified")
	}
	msg, err := m.inventory.Update(stock.Consumable, stock, opts...)
	return castStock(msg, err)
}

func (m *Model) DeleteStock(consumable string, opts ...resource.WriteOption) (*Consumable_Stock, error) {
	return castStock(m.inventory.Delete(consumable, opts...))
}

type StockChange struct {
	ChangeTime time.Time
	Value      *Consumable_Stock
}

// PullStock subscribes to changes in a single consumables stock.
// The returned channel will be closed if ctx is Done or the stock record identified by consumable is deleted.
func (m *Model) PullStock(ctx context.Context, consumable string, opts ...resource.ReadOption) <-chan StockChange {
	send := make(chan StockChange)
	go func() {
		defer close(send)
		for change := range m.inventory.PullID(ctx, consumable, opts...) {
			select {
			case <-ctx.Done():
				return
			case send <- StockChange{ChangeTime: change.ChangeTime, Value: change.Value.(*Consumable_Stock)}:
			}
		}
	}()
	return send
}

// ListInventory returns all known stock records.
func (m *Model) ListInventory(opts ...resource.ReadOption) []*Consumable_Stock {
	msgs := m.inventory.List(opts...)
	res := make([]*Consumable_Stock, len(msgs))
	for i, msg := range msgs {
		res[i] = msg.(*Consumable_Stock)
	}
	return res
}

type InventoryChange struct {
	ID         string
	ChangeTime time.Time
	ChangeType typespb.ChangeType
	OldValue   *Consumable_Stock
	NewValue   *Consumable_Stock
}

// PullInventory subscribes to changes in the list of known stock records.
func (m *Model) PullInventory(ctx context.Context, opts ...resource.ReadOption) <-chan InventoryChange {
	send := make(chan InventoryChange)
	go func() {
		defer close(send)
		for change := range m.inventory.Pull(ctx, opts...) {
			oldVal, newVal := castStockChange(change)
			event := InventoryChange{
				ID:         change.Id,
				ChangeTime: change.ChangeTime,
				ChangeType: change.ChangeType,
				OldValue:   oldVal,
				NewValue:   newVal,
			}
			select {
			case <-ctx.Done():
				return
			case send <- event:
			}
		}
	}()
	return send
}

// DispenseInstantly removes quantity amount from the stock of the named consumable.
// This updates the stock LastDispensed, Used, and Remaining. Used and Remaining are only updated if they have a
// (possibly zero) value in the stock already.
// Stock Used and Remaining units are maintained, the quantity is converted to those units before modification.
func (m *Model) DispenseInstantly(consumable string, quantity *Consumable_Quantity) (*Consumable_Stock, error) {
	var maskedErr error // for tracking errors in interceptors
	stock, err := m.UpdateStock(&Consumable_Stock{Consumable: consumable}, resource.InterceptBefore(func(old, new proto.Message) {
		oldVal := old.(*Consumable_Stock)
		newVal := new.(*Consumable_Stock)
		if err := updateStock(quantity, oldVal, newVal); err != nil {
			// on error we don't want to make any changes to the stock value, reset things
			maskedErr = err
			proto.Reset(newVal)
			proto.Merge(newVal, oldVal)
			return
		}
		newVal.LastDispensed = quantity
		newVal.Dispensing = false
	}))
	if err != nil {
		return nil, err
	}
	if maskedErr != nil {
		return nil, err
	}
	return stock, nil
}

func updateStock(quantity *Consumable_Quantity, src, dst *Consumable_Stock) error {
	if src.Used != nil {
		delta, err := ConvertUnit32(quantity.Amount, quantity.Unit, src.Used.Unit)
		if err != nil {
			return err
		}
		dst.Used = &Consumable_Quantity{Unit: src.Used.Unit, Amount: src.Used.Amount + delta}
	}
	if src.Remaining != nil {
		delta, err := ConvertUnit32(quantity.Amount, quantity.Unit, src.Remaining.Unit)
		if err != nil {
			return err
		}
		amount := src.Remaining.Amount - delta
		if amount < 0 {
			amount = 0
		}
		dst.Remaining = &Consumable_Quantity{Unit: src.Used.Unit, Amount: amount}
	}
	return nil
}

func castConsumable(msg proto.Message, err error) (*Consumable, error) {
	if msg == nil {
		return nil, err
	}
	return msg.(*Consumable), err
}

func castConsumableChange(change *resource.CollectionChange) (oldVal, newVal *Consumable) {
	if change.OldValue != nil {
		oldVal = change.OldValue.(*Consumable)
	}
	if change.NewValue != nil {
		newVal = change.NewValue.(*Consumable)
	}
	return
}

func castStock(msg proto.Message, err error) (*Consumable_Stock, error) {
	if msg == nil {
		return nil, err
	}
	return msg.(*Consumable_Stock), err
}

func castStockChange(change *resource.CollectionChange) (oldVal, newVal *Consumable_Stock) {
	if change.OldValue != nil {
		oldVal = change.OldValue.(*Consumable_Stock)
	}
	if change.NewValue != nil {
		newVal = change.NewValue.(*Consumable_Stock)
	}
	return
}
