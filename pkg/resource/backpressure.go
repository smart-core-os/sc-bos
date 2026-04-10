package resource

import (
	"container/list"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
)

// mergeCollectionExcess acts on a chan of *CollectionChange combining changes with the same key to maintain the
// semantics without needing to emit every events.
// This will use memory proportional to one change for each id that has not been emitted yet.
func mergeCollectionExcess(in <-chan *CollectionChange) <-chan *CollectionChange {
	out := make(chan *CollectionChange)
	go func() {
		defer close(out)

		messages := make(map[string]CollectionChange)
		var queue list.List // of string, Front is which id to send next
		event := func() *CollectionChange {
			if queue.Len() == 0 {
				return nil
			}
			id := queue.Front().Value.(string)
			change := messages[id]
			return &change
		}

		for {
			if queue.Len() > 0 {
				select {
				case newAny, ok := <-in:
					if !ok {
						return
					}
					newMessage := *newAny
					oldMessage, hasOld := messages[newMessage.Id]
					id := newMessage.Id
					if hasOld {
						var send bool
						newMessage, send = mergeChanges(oldMessage, newMessage)
						for n := queue.Front(); n != nil; n = n.Next() {
							if n.Value.(string) == id {
								queue.Remove(n)
								break
							}
						}
						if !send {
							delete(messages, id)
							continue
						}
					}

					messages[id] = newMessage
					queue.PushBack(id)
				case out <- event():
					front := queue.Front()
					queue.Remove(front)
					delete(messages, front.Value.(string))
				}
			} else {
				newAny, ok := <-in
				if !ok {
					return
				}
				newMessage := *newAny
				messages[newMessage.Id] = newMessage
				queue.PushBack(newMessage.Id)
			}
		}

	}()
	return out
}

func mergeChanges(a, b CollectionChange) (c CollectionChange, send bool) {
	b.LastSeedValue = a.LastSeedValue || b.LastSeedValue

	switch a.ChangeType {
	case typespb.ChangeType_ADD:
		switch b.ChangeType {
		case typespb.ChangeType_ADD: // not sure how this happens, but sure
			return b, true
		case typespb.ChangeType_UPDATE, typespb.ChangeType_REPLACE:
			b.ChangeType = typespb.ChangeType_ADD
			b.OldValue = nil
			return b, true
		case typespb.ChangeType_REMOVE:
			return CollectionChange{}, false
		default:
			return b, true
		}
	case typespb.ChangeType_UPDATE:
		b.OldValue = a.OldValue
		if b.ChangeType == typespb.ChangeType_ADD { // not sure how this happens, but sure
			b.ChangeType = typespb.ChangeType_REPLACE
		}
		return b, true
	case typespb.ChangeType_REPLACE:
		b.OldValue = a.OldValue
		if ct := b.ChangeType; ct == typespb.ChangeType_ADD || ct == typespb.ChangeType_UPDATE {
			b.ChangeType = typespb.ChangeType_REPLACE
		}
		return b, true
	case typespb.ChangeType_REMOVE:
		b.OldValue = a.OldValue
		if b.ChangeType != typespb.ChangeType_REMOVE {
			b.ChangeType = typespb.ChangeType_REPLACE
		}
		return b, true
	default:
		return b, true
	}
}
