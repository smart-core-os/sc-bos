package cmp

import (
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

func And(eqs ...Message) Message {
	return func(x, y proto.Message) bool {
		for _, eq := range eqs {
			if !eq(x, y) {
				return false
			}
		}
		return true
	}
}

func ValueAnd(eqs ...Value) Value {
	return func(fd pref.FieldDescriptor, x, y pref.Value) (equal bool, ok bool) {
		for _, eq := range eqs {
			if equal, ok2 := eq(fd, x, y); ok2 {
				ok = ok2
				if !equal {
					return false, true
				}
			}
		}
		return true, ok
	}
}

func Or(eqs ...Message) Message {
	return func(x, y proto.Message) bool {
		for _, eq := range eqs {
			if eq(x, y) {
				return true
			}
		}
		return false
	}
}

func ValueOr(eqs ...Value) Value {
	return func(fd pref.FieldDescriptor, x, y pref.Value) (equal bool, ok bool) {
		for _, eq := range eqs {
			if equal, ok2 := eq(fd, x, y); ok2 {
				ok = ok2
				if equal {
					return true, true
				}
			}
		}
		return false, ok
	}
}
