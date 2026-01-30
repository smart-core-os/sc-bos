package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/smart-core-os/gobacnet/property"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/gobacnet/types/objecttype"
)

type ObjectID bactypes.ObjectID

//goland:noinspection GoMixedReceiverTypes
func (o ObjectID) String() string {
	return fmt.Sprintf("%s:%d", o.Type, o.Instance)
}

//goland:noinspection GoMixedReceiverTypes
func (o ObjectID) MarshalJSON() ([]byte, error) {
	if objecttype.Known(o.Type) {
		return json.Marshal(fmt.Sprintf("%s:%d", o.Type, o.Instance))
	} else {
		return json.Marshal(fmt.Sprintf("%d:%d", o.Type, o.Instance))
	}
}

//goland:noinspection GoMixedReceiverTypes
func (o *ObjectID) UnmarshalJSON(bytes []byte) error {
	var str string
	if err := json.Unmarshal(bytes, &str); err != nil {
		return err
	}

	oid, err := ObjectIDFromString(str)
	if err != nil {
		return err
	}
	*o = oid
	return nil
}

func ObjectIDFromString(str string) (ObjectID, error) {
	parts := strings.SplitN(str, ":", 2)
	if len(parts) != 2 {
		return ObjectID{}, fmt.Errorf("expecting {type}:{instance}, got %s", str)
	}
	typeStr, idStr := parts[0], parts[1]
	idInt, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		return ObjectID{}, fmt.Errorf("instance is not an int: %w", err)
	}
	typeId, err := strconv.ParseInt(typeStr, 10, 16)
	if err == nil {
		return ObjectID{
			Type:     bactypes.ObjectType(typeId),
			Instance: bactypes.ObjectInstance(idInt),
		}, nil
	}

	fromString, ok := objecttype.FromString(typeStr)
	if !ok {
		return ObjectID{}, fmt.Errorf("object type is unknown and not an int %w", err)
	}
	return ObjectID{
		Type:     fromString,
		Instance: bactypes.ObjectInstance(idInt),
	}, nil
}

type PropertyID property.ID

//goland:noinspection GoMixedReceiverTypes
func (p PropertyID) String() string {
	return property.ID(p).String()
}

//goland:noinspection GoMixedReceiverTypes
func (p PropertyID) MarshalJSON() ([]byte, error) {
	if property.Known(property.ID(p)) {
		return json.Marshal(property.ID(p).String())
	} else {
		return json.Marshal(int(p))
	}
}

//goland:noinspection GoMixedReceiverTypes
func (p *PropertyID) UnmarshalJSON(bytes []byte) error {
	var str string
	var num uint32

	strErr := json.Unmarshal(bytes, &str)
	if strErr == nil {
		id, ok := property.FromString(str)
		if !ok {
			return fmt.Errorf("unknown property name %s", str)
		}
		*p = PropertyID(id)
		return nil
	}

	numErr := json.Unmarshal(bytes, &num)
	if numErr == nil {
		*p = PropertyID(num)
		return nil
	}

	// both strErr and numErr are not nil
	return fmt.Errorf("not a string or number")
}
