package wrap

import (
	"google.golang.org/grpc"
)

// Unwrapper defines the Unwrap method for exposing underlying implementation types.
type Unwrapper interface {
	Unwrap() any
}

type ServiceUnwrapper interface {
	UnwrapService() (grpc.ClientConnInterface, grpc.ServiceDesc)
}

// UnwrapFully repeatedly casts then unwraps obj until obj no longer implements Unwrapper.
func UnwrapFully(obj any) any {
	for t, ok := obj.(Unwrapper); ok; t, ok = obj.(Unwrapper) {
		obj = t.Unwrap()
	}
	return obj
}
