package vendingpb

import (
	"fmt"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// DefaultModelOptions holds the default options for the model.
var DefaultModelOptions []resource.Option

// ModelOption defined the base type for all options that apply to this traits model.
type ModelOption interface {
	resource.Option
	applyModel(args *modelArgs)
}

// WithInventoryOption configures the inventory resource of the model.
func WithInventoryOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.inventoryOptions = append(args.inventoryOptions, opts...)
	})
}

// WithInitialStock configures the model to initialise with the given stock.
// Can be used multiple times with stock being additive.
// Creating a model with duplicate stock consumable names will panic.
// Calling this function with an empty stock consumable property will panic.
func WithInitialStock(inventory ...*traits.Consumable_Stock) resource.Option {
	opts := make([]resource.Option, len(inventory))
	for i, stock := range inventory {
		if stock.Consumable == "" {
			panic(fmt.Sprintf("stock at index %v has no Consumable property", i))
		}
		opts[i] = resource.WithInitialRecord(stock.Consumable, stock)
	}
	return WithInventoryOption(opts...)
}

// WithConsumablesOption configures the consumables resource of the model.
func WithConsumablesOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.inventoryOptions = append(args.inventoryOptions, opts...)
	})
}

// WithInitialConsumable configures the model to initialise with the given consumable.
// Can be used multiple times with consumables being additive.
// Creating a model with duplicate consumable names will panic.
// Calling this function with an empty consumable name property will panic.
func WithInitialConsumable(consumables ...*traits.Consumable) resource.Option {
	opts := make([]resource.Option, len(consumables))
	for i, consumable := range consumables {
		if consumable.Name == "" {
			panic(fmt.Sprintf("consumable at index %v has no Name property", i))
		}
		opts[i] = resource.WithInitialRecord(consumable.Name, consumable)
	}
	return WithConsumablesOption(opts...)
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	consumableOptions []resource.Option
	inventoryOptions  []resource.Option
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(ModelOption); ok {
			v.applyModel(a)
			continue
		}
		a.consumableOptions = append(a.consumableOptions, opt)
		a.inventoryOptions = append(a.inventoryOptions, opt)
	}
}

func modelOptionFunc(fn func(args *modelArgs)) ModelOption {
	return modelOption{resource.EmptyOption{}, fn}
}

type modelOption struct {
	resource.Option
	fn func(args *modelArgs)
}

func (m modelOption) applyModel(args *modelArgs) {
	m.fn(args)
}
