package publicationpb

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

// WithPublicationOption configures the publication resource of the model.
func WithPublicationOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.publicationsOptions = append(args.publicationsOptions, opts...)
	})
}

// WithInitialPublication configures the model to initialise with the given publication.
// Can be used multiple times with publications being additive.
// Creating a model with duplicate publication IDs will panic.
// Calling this function with an empty publication id property will panic.
func WithInitialPublication(publications ...*traits.Publication) resource.Option {
	opts := make([]resource.Option, len(publications))
	for i, publication := range publications {
		if publication.Id == "" {
			panic(fmt.Sprintf("publication at index %v has no Id property", i))
		}
		opts[i] = resource.WithInitialRecord(publication.Id, publication)
	}
	return WithPublicationOption(opts...)
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	publicationsOptions []resource.Option
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(ModelOption); ok {
			v.applyModel(a)
			continue
		}
		a.publicationsOptions = append(a.publicationsOptions, opt)
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

// WriteOption allow extended configuration of write operations for Model.
type WriteOption interface {
	resource.WriteOption
	applyModel(args *writeArgs)
}

type writeArgs struct {
	newPublishTime bool // causes a publish time to be minted
	newVersion     bool // causes a version to be minted
	resetReceipt   bool // clear receipt properties (but not audience)
}

func calcWriteArgs(opts ...resource.WriteOption) writeArgs {
	args := new(writeArgs)
	for _, opt := range opts {
		if wo, ok := opt.(WriteOption); ok {
			wo.applyModel(args)
		}
	}
	return *args
}

type writeOption struct {
	resource.WriteOption
	fn func(args *writeArgs)
}

func (w writeOption) applyModel(args *writeArgs) {
	w.fn(args)
}

func writeOptionFunc(fn func(args *writeArgs)) WriteOption {
	return writeOption{resource.EmptyWriteOption{}, fn}
}

// WithNewPublishTime instructs the write operation to calculate a new publish time for the publication.
func WithNewPublishTime() resource.WriteOption {
	return writeOptionFunc(func(args *writeArgs) {
		args.newPublishTime = true
	})
}

// WithNewVersion instructs a write operation to calculate a new version for the publication.
func WithNewVersion() resource.WriteOption {
	return writeOptionFunc(func(args *writeArgs) {
		args.newVersion = true
	})
}

// WithResetReceipt instructs a write operation to reset receipt properties if they exist in the audience for a
// publication.
func WithResetReceipt() resource.WriteOption {
	return writeOptionFunc(func(args *writeArgs) {
		args.resetReceipt = true
	})
}
