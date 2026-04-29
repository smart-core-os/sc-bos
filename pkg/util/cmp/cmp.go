package cmp

import (
	"bytes"
	"math"
	"reflect"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

type Message func(x, y proto.Message) bool

type Value func(fd pref.FieldDescriptor, x, y pref.Value) (equal bool, ok bool)

// Equal returns a Message that compares like proto.Equal(x, y) except where value comparisons match instead.
func Equal(cmpValue ...Value) Message {
	eq := equator{cmpValue: ValueAnd(cmpValue...)}
	return eq.compare
}

type equator struct {
	cmpValue Value
}

func (eq equator) compare(x, y proto.Message) bool {
	// this code is heavily borrowed from proto.Equal, but adjusted so we can compare different types
	if x == nil || y == nil {
		return x == nil && y == nil
	}
	mx, my := x.ProtoReflect(), y.ProtoReflect()
	if mx.IsValid() != my.IsValid() {
		return false
	}
	return eq.equalMessage(mx, my)
}

// equalMessage compares two messages.
func (eq equator) equalMessage(mx, my pref.Message) bool {
	if mx.Descriptor() != my.Descriptor() {
		return false
	}

	nx := 0
	equal := true
	mx.Range(func(fd pref.FieldDescriptor, vx pref.Value) bool {
		nx++
		vy := my.Get(fd)
		equal = my.Has(fd) && eq.equalField(fd, vx, vy)
		return equal
	})
	if !equal {
		return false
	}
	ny := 0
	my.Range(func(fd pref.FieldDescriptor, vx pref.Value) bool {
		ny++
		return true
	})
	if nx != ny {
		return false
	}

	return eq.equalUnknown(mx.GetUnknown(), my.GetUnknown())
}

// equalField compares two fields.
func (eq equator) equalField(fd pref.FieldDescriptor, x, y pref.Value) bool {
	switch {
	// This is the case we've added, ignore PullResponse.Change.change_time
	case fd.Name() == "change_time" && fd.ContainingMessage().Name() == "Change":
		return true
	case fd.IsList():
		return eq.equalList(fd, x.List(), y.List())
	case fd.IsMap():
		return eq.equalMap(fd, x.Map(), y.Map())
	default:
		return eq.equalValue(fd, x, y)
	}
}

// equalMap compares two maps.
func (eq equator) equalMap(fd pref.FieldDescriptor, x, y pref.Map) bool {
	if x.Len() != y.Len() {
		return false
	}
	equal := true
	x.Range(func(k pref.MapKey, vx pref.Value) bool {
		vy := y.Get(k)
		equal = y.Has(k) && eq.equalValue(fd.MapValue(), vx, vy)
		return equal
	})
	return equal
}

// equalList compares two lists.
func (eq equator) equalList(fd pref.FieldDescriptor, x, y pref.List) bool {
	if x.Len() != y.Len() {
		return false
	}
	for i := x.Len() - 1; i >= 0; i-- {
		if !eq.equalValue(fd, x.Get(i), y.Get(i)) {
			return false
		}
	}
	return true
}

// equalValue compares two singular values.
func (eq equator) equalValue(fd pref.FieldDescriptor, x, y pref.Value) bool {
	if eq.cmpValue != nil {
		if equal, ok := eq.cmpValue(fd, x, y); ok {
			return equal
		}
	}
	switch fd.Kind() {
	case pref.BoolKind:
		return x.Bool() == y.Bool()
	case pref.EnumKind:
		return x.Enum() == y.Enum()
	case pref.Int32Kind, pref.Sint32Kind,
		pref.Int64Kind, pref.Sint64Kind,
		pref.Sfixed32Kind, pref.Sfixed64Kind:
		return x.Int() == y.Int()
	case pref.Uint32Kind, pref.Uint64Kind,
		pref.Fixed32Kind, pref.Fixed64Kind:
		return x.Uint() == y.Uint()
	case pref.FloatKind, pref.DoubleKind:
		fx := x.Float()
		fy := y.Float()
		if math.IsNaN(fx) || math.IsNaN(fy) {
			return math.IsNaN(fx) && math.IsNaN(fy)
		}
		return fx == fy
	case pref.StringKind:
		return x.String() == y.String()
	case pref.BytesKind:
		return bytes.Equal(x.Bytes(), y.Bytes())
	case pref.MessageKind, pref.GroupKind:
		return eq.equalMessage(x.Message(), y.Message())
	default:
		return x.Interface() == y.Interface()
	}
}

// equalUnknown compares unknown fields by direct comparison on the raw bytes
// of each individual field number.
func (eq equator) equalUnknown(x, y pref.RawFields) bool {
	if len(x) != len(y) {
		return false
	}
	if bytes.Equal([]byte(x), []byte(y)) {
		return true
	}

	mx := make(map[pref.FieldNumber]pref.RawFields)
	my := make(map[pref.FieldNumber]pref.RawFields)
	for len(x) > 0 {
		fnum, _, n := protowire.ConsumeField(x)
		mx[fnum] = append(mx[fnum], x[:n]...)
		x = x[n:]
	}
	for len(y) > 0 {
		fnum, _, n := protowire.ConsumeField(y)
		my[fnum] = append(my[fnum], y[:n]...)
		y = y[n:]
	}
	return reflect.DeepEqual(mx, my)
}
