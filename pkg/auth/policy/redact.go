package policy

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const redactedString = "[REDACTED]"

// redacted returns a copy of msg with every field marked [debug_redact = true] removed, at any depth.
// A redacted string is replaced with "[REDACTED]" so the log shows it was set, other fields are cleared.
func redacted(msg proto.Message) proto.Message {
	msg = proto.Clone(msg)
	redact(msg.ProtoReflect())
	return msg
}

func redact(m protoreflect.Message) {
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if opts, ok := fd.Options().(*descriptorpb.FieldOptions); ok && opts.GetDebugRedact() {
			if fd.Kind() == protoreflect.StringKind && !fd.IsList() {
				m.Set(fd, protoreflect.ValueOfString(redactedString))
			} else {
				m.Clear(fd)
			}
			return true
		}
		switch {
		case fd.IsMap():
			if fd.MapValue().Message() != nil {
				v.Map().Range(func(_ protoreflect.MapKey, v protoreflect.Value) bool {
					redact(v.Message())
					return true
				})
			}
		case fd.IsList():
			if fd.Message() != nil {
				l := v.List()
				for i := range l.Len() {
					redact(l.Get(i).Message())
				}
			}
		case fd.Message() != nil:
			redact(v.Message())
		}
		return true
	})
}
