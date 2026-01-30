package mdblock

import (
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/block"
)

// Categories contains a block for each top level metadata category: location, membership, etc.
var Categories = func() []block.Block {
	msg := (&traits.Metadata{}).ProtoReflect()
	desc := msg.Descriptor()
	fields := desc.Fields()
	blocks := make([]block.Block, 0, fields.Len())
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		blocks = append(blocks, block.Block{
			Path: []string{field.JSONName()},
		})
	}
	return blocks
}()
