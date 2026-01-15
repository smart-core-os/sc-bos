package node

import (
	"fmt"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/internal/node/nodeopts"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait/metadatapb"
)

// mergeAllMetadata uses the metadatapb.Merge algorithm to merge multiple metadata objects into one.
// The input metadata objects will not be modified.
// Metadata at higher indexes in the mds slice will override metadata at lower indexes.
func mergeAllMetadata(name string, mds ...*traits.Metadata) *traits.Metadata {
	if len(mds) == 0 {
		return nil
	}
	if len(mds) == 1 {
		md := mds[0]
		md.Name = name
		// for consistency, sort the traits by name
		slices.SortFunc(md.Traits, func(a, b *traits.TraitMetadata) int {
			return strings.Compare(a.Name, b.Name)
		})
		return md
	}

	md := &traits.Metadata{
		Name: name,
	}
	for _, m := range mds {
		m = proto.Clone(m).(*traits.Metadata)
		metadatapb.Merge(md, m)
		md = m
	}
	return md
}

// metadataList is a list of metadata entries, each with a unique, strictly increasing, ID.
type metadataList []metadataEntry

type metadataEntry struct {
	id uint64
	md *traits.Metadata
}

func (ml *metadataList) add(md *traits.Metadata) uint64 {
	if len(*ml) == 0 {
		*ml = append(*ml, metadataEntry{id: 0, md: md})
		return 0
	}
	id := (*ml)[len(*ml)-1].id + 1
	*ml = append(*ml, metadataEntry{id: id, md: md})
	return id
}

func (ml *metadataList) remove(id uint64) {
	i, found := slices.BinarySearchFunc(*ml, id, func(e metadataEntry, id uint64) int {
		if e.id < id {
			return -1
		} else if e.id > id {
			return 1
		}
		return 0
	})
	if !found {
		return
	}
	*ml = slices.Replace(*ml, i, i+1)
}

func (ml *metadataList) merge() *traits.Metadata {
	if len(*ml) == 0 {
		return nil
	}
	if len(*ml) == 1 {
		return (*ml)[0].md
	}

	name := ""
	mds := make([]*traits.Metadata, len(*ml))
	for i, entry := range *ml {
		mds[i] = entry.md
		if name == "" && entry.md.Name != "" {
			name = entry.md.Name
		}
	}
	return mergeAllMetadata(name, mds...)
}

func (ml *metadataList) isEmpty() bool {
	return len(*ml) == 0
}

func (ml *metadataList) updateCollection(c nodeopts.Store, opts ...resource.WriteOption) error {
	if ml.isEmpty() {
		return fmt.Errorf("no name: empty metadata list")
	}
	name := (*ml)[0].md.Name
	if name == "" {
		return fmt.Errorf("no name: list item has no name")
	}
	md := ml.merge()
	opts = append(opts, resource.WithUpdatePaths("name", "metadata"), resource.InterceptAfter(func(old, new proto.Message) {
		newDevice := new.(*devicespb.Device)
		newDevice.Metadata = md
	}))
	_, err := c.Update(&devicespb.Device{Name: name}, opts...)
	return err
}
