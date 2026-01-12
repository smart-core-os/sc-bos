package token

import (
	"encoding/json"
	"errors"

	"github.com/smart-core-os/sc-bos/internal/auth/permission"
	"github.com/smart-core-os/sc-bos/pkg/proto/accountpb"
)

type Claims struct {
	SystemRoles []string               `json:"system_roles"` // The built-in system roles that this token is authorized for
	IsService   bool                   `json:"is_service"`   // True if the subject is an application acting on its own behalf, false if it's a user
	Permissions []PermissionAssignment `json:"permissions"`
}

type PermissionAssignment struct {
	Permission   permission.ID `json:"permission"`    // The name of the permission, e.g. trait:read:*
	Scoped       bool          `json:"scoped"`        // True if the permission is scoped to a specific resource
	ResourceType ResourceType  `json:"resource_type"` // The type of resource this permission is scoped to
	Resource     string        `json:"resource"`      // The resource this permission is scoped to - its meaning depends on the resource type
}

type ResourceType accountpb.RoleAssignment_ResourceType

func ParseResourceType(s string) (ResourceType, bool) {
	rt, ok := accountpb.RoleAssignment_ResourceType_value[s]
	if !ok {
		return 0, false
	}
	return ResourceType(rt), true
}

//goland:noinspection GoMixedReceiverTypes
func (rt ResourceType) MarshalJSON() ([]byte, error) {
	rtp := accountpb.RoleAssignment_ResourceType(rt)
	desc := rtp.Descriptor().Values().ByNumber(rtp.Number())
	if desc == nil {
		return json.Marshal(int32(rtp.Number()))
	} else {
		return json.Marshal(string(desc.Name()))
	}
}

//goland:noinspection GoMixedReceiverTypes
func (rt *ResourceType) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return errors.New("want string or int")
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		parsed, ok := ParseResourceType(s)
		if !ok {
			return errors.New("invalid resource type: " + s)
		}
		*rt = parsed
		return nil
	} else {
		var i int32
		if err := json.Unmarshal(b, &i); err != nil {
			return err
		}
		*rt = ResourceType(i)
		return nil
	}
}

//goland:noinspection GoMixedReceiverTypes
func (rt ResourceType) String() string {
	return accountpb.RoleAssignment_ResourceType(rt).String()
}
