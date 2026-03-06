package router

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

func TestCopyRecver_RecvMsg(t *testing.T) {
	moreMap := func(more ...string) map[string]string {
		if len(more) == 0 {
			return nil
		}
		m := make(map[string]string, len(more)/2)
		for i := 0; i < len(more); i += 2 {
			m[more[i]] = more[i+1]
		}
		return m
	}
	msg := func(s string, more ...string) proto.Message {
		return &traits.Metadata{Name: s, More: moreMap(more...)}
	}
	msg2 := func(s string, more ...string) proto.Message {
		return &traits.TraitMetadata{Name: s, More: moreMap(more...)}
	}
	// simulate descriptors from the reflection package
	mkDynamic := func(m proto.Message) proto.Message {
		// convert the descriptor to an equivalent one via the proto file representation
		descriptor := m.ProtoReflect().Descriptor()
		file, err := protodesc.NewFile(protodesc.ToFileDescriptorProto(descriptor.ParentFile()), protoregistry.GlobalFiles)
		if err != nil {
			t.Fatalf("NewFile() error = %v", err)
		}
		msgDesc := file.Messages().ByName(descriptor.Name())
		if msgDesc == nil {
			t.Fatalf("message not found in file: %v", descriptor.Name())
		}
		dm := dynamicpb.NewMessage(msgDesc)

		// copy over the data, don't use proto.Merge!
		data, err := proto.Marshal(m)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		if err := proto.Unmarshal(data, dm); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		return dm
	}

	tests := []struct {
		name               string
		src, dst, want     proto.Message
		wantErr, wantPanic bool
	}{
		{"msg,msg", msg("a"), msg("b", "k", "v"), msg("a", "k", "v"), false, false},
		{"msg,dynamic", msg("a"), mkDynamic(msg("b", "k", "v")), msg("a", "k", "v"), false, false},
		{"dynamic,msg", mkDynamic(msg("a")), msg("b", "k", "v"), msg("a", "k", "v"), false, false},
		{"dynamic,msg", mkDynamic(msg("a")), msg("b", "k", "v"), msg("a", "k", "v"), false, false},
		{"dynamic,dynamic", mkDynamic(msg("a")), mkDynamic(msg("b", "k", "v")), msg("a", "k", "v"), false, false},
		{"msg1,msg2", msg("1"), msg2("2"), msg2("2"), false, true},
		{"msg2,msg1", msg2("2"), msg("1"), msg("1"), false, true},
		{"msg1,dynamic2", msg("1"), mkDynamic(msg2("2")), msg2("2"), false, true},
		{"dynamic2,msg1", mkDynamic(msg2("2")), msg("1"), msg("1"), false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r != nil {
					if !tt.wantPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				} else if tt.wantPanic {
					t.Error("expected panic")
				}
			}()

			cr := copyRecver{from: tt.src}
			if err := cr.RecvMsg(tt.dst); (err != nil) != tt.wantErr {
				t.Errorf("RecvMsg() error = %v, wantErr %v", err, tt.wantErr)
			}

			// proto.Equal doesn't work for messages with different descriptors
			wantBytes, _ := proto.MarshalOptions{Deterministic: true}.Marshal(tt.want)
			gotBytes, _ := proto.MarshalOptions{Deterministic: true}.Marshal(tt.dst)
			if !bytes.Equal(wantBytes, gotBytes) {
				t.Errorf("RecvMsg() = %v, want %v", tt.dst, tt.want)
			}
		})
	}
}
