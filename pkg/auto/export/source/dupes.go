package source

import (
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
)

type duplicates struct {
	sent map[string]proto.Message
	cmp  cmp.Message
}

func trackDuplicates(cmp cmp.Message) *duplicates {
	return &duplicates{
		sent: make(map[string]proto.Message),
		cmp:  cmp,
	}
}

func allowDuplicates() *duplicates {
	return nil
}

func (d *duplicates) Changed(key string, other proto.Message) (commit func(), ok bool) {
	if d == nil {
		return func() {}, true
	}

	old, ok := d.sent[key]
	if !ok {
		return func() { d.sent[key] = other }, true
	}

	if d.cmp(old, other) {
		return func() {}, false
	}
	return func() { d.sent[key] = other }, true
}
