package resource

import (
	"container/list"

	"github.com/smart-core-os/sc-api/go/types"
)

// mergeCollectionExcess acts on a chan of *CollectionChange combining changes with the same key to maintain the
// semantics without needing to emit every events.
// This will use memory proportional to one change for each id that has not been emitted yet.
func mergeCollectionExcess(in <-chan any) <-chan any {
	out := make(chan any)
	go func() {
		defer close(out)

		messages := make(map[string]CollectionChange)
		var queue list.List // of string, Front is which id to send next
		event := func() any {
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
					newMessage := *(newAny.(*CollectionChange))
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
				newMessage := *(newAny.(*CollectionChange))
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
	case types.ChangeType_ADD:
		switch b.ChangeType {
		case types.ChangeType_ADD: // not sure how this happens, but sure
			return b, true
		case types.ChangeType_UPDATE, types.ChangeType_REPLACE:
			b.ChangeType = types.ChangeType_ADD
			b.OldValue = nil
			return b, true
		case types.ChangeType_REMOVE:
			return CollectionChange{}, false
		default:
			return b, true
		}
	case types.ChangeType_UPDATE:
		b.OldValue = a.OldValue
		if b.ChangeType == types.ChangeType_ADD { // not sure how this happens, but sure
			b.ChangeType = types.ChangeType_REPLACE
		}
		return b, true
	case types.ChangeType_REPLACE:
		b.OldValue = a.OldValue
		if ct := b.ChangeType; ct == types.ChangeType_ADD || ct == types.ChangeType_UPDATE {
			b.ChangeType = types.ChangeType_REPLACE
		}
		return b, true
	case types.ChangeType_REMOVE:
		b.OldValue = a.OldValue
		if b.ChangeType != types.ChangeType_REMOVE {
			b.ChangeType = types.ChangeType_REPLACE
		}
		return b, true
	default:
		return b, true
	}
}
