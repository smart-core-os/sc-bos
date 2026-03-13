package resource

import (
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// CreateFn is called to generate a message based on the ID the message is going to have.
type CreateFn func(id string) proto.Message

// GetFn is called to retrieve the message from the external store.
type GetFn func() (item proto.Message, err error)

// ChangeFn is called to apply changes to a proto.Message.
// old and dst are the current message and a message to change.
// If old is nil, there is no current message.
// If dst is nil, then the resource was unable to allocate a message of the correct type to apply changes to.
// The implementation is free to ignore both arguments but must return a message containing the desired changes.
type ChangeFn func(old, dst proto.Message) (proto.Message, error)

// SaveFn is called to save the message in the external store.
type SaveFn func(msg proto.Message)

// GetAndUpdate applies an atomic get and update operation in the context of proto messages.
// mu.RLock will be held during the get call.
// mu.Lock will be held during the save call.
// No locks will be held during the change call.
//
// An error will be returned if the value returned by get changes during the change call.
func GetAndUpdate(mu *sync.RWMutex, get GetFn, change ChangeFn, save SaveFn) (oldValue proto.Message, newValue proto.Message, err error) {
	mu.RLock()
	oldValue, err = get()
	mu.RUnlock()
	if err != nil {
		return nil, nil, err
	}

	newValue = proto.Clone(oldValue)
	if newValue, err = change(oldValue, newValue); err != nil {
		return oldValue, newValue, err
	}

	mu.Lock()
	defer mu.Unlock()
	oldValueAgain, _ := get()
	if !proto.Equal(oldValue, oldValueAgain) {
		return oldValue, newValue, status.Errorf(codes.Aborted, "concurrent update detected")
	}

	save(newValue)
	return oldValue, newValue, nil
}
