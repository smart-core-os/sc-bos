package config

import (
	"testing"

	"github.com/smart-core-os/gobacnet/types/objecttype"
)

func TestObjectRef_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		want    ObjectRef
		wantErr bool
	}{
		{`""`, ObjectRef{}, false},
		{`"some string"`, ObjectRef{name: "some string"}, false},
		{`"AnalogValue:12"`, ObjectRef{id: ObjectID{Type: objecttype.AnalogValue, Instance: 12}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ObjectRef
			if err := (&got).UnmarshalJSON([]byte(tt.name)); (err != nil) != tt.wantErr {
				t.Errorf("ObjectRef.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ObjectRef.UnmarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
