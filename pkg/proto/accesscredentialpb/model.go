package accesscredentialpb

import (
	"fmt"
	"slices"
	"strconv"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
)

// Model is an in-memory store of the credentials held by a single cardholder.
type Model struct {
	support *CredentialSupport

	mu        sync.Mutex
	byID      map[string]*Credential
	lastID    int
	lastValue int
}

// NewModel creates a Model that accepts the credential types in support.
// If support is nil, DefaultSupport is used.
func NewModel(support *CredentialSupport) *Model {
	if support == nil {
		support = DefaultSupport()
	}
	return &Model{
		support: support,
		byID:    make(map[string]*Credential),
	}
}

// DefaultSupport returns a CredentialSupport describing a few common credential types.
func DefaultSupport() *CredentialSupport {
	return &CredentialSupport{
		ResourceSupport: &typespb.ResourceSupport{
			Readable: true,
			Writable: true,
			WritableFields: &fieldmaskpb.FieldMask{Paths: []string{
				"type", "value", "state", "active_time", "expire_time", "issue_level", "invitation", "more",
			}},
		},
		Types: []*CredentialType{
			{
				Id: "card", DisplayName: "Card", Kind: Credential_CARD,
				ValueSource:    CredentialType_EITHER,
				WritableStates: []Credential_State{Credential_ACTIVE, Credential_DISABLED, Credential_LOST, Credential_STOLEN},
			},
			{
				Id: "fob", DisplayName: "Fob", Kind: Credential_FOB,
				ValueSource:    CredentialType_CALLER_SUPPLIED,
				WritableStates: []Credential_State{Credential_ACTIVE, Credential_DISABLED, Credential_LOST, Credential_STOLEN},
			},
			{
				Id: "mobile", DisplayName: "Mobile", Kind: Credential_MOBILE,
				ValueSource:        CredentialType_SYSTEM_ALLOCATED,
				WritableStates:     []Credential_State{Credential_ACTIVE, Credential_DISABLED},
				InvitationRequired: true,
			},
			{
				Id: "plate", DisplayName: "Number plate", Kind: Credential_VEHICLE_PLATE,
				ValueSource:    CredentialType_CALLER_SUPPLIED,
				WritableStates: []Credential_State{Credential_ACTIVE, Credential_DISABLED},
			},
		},
	}
}

// DescribeCredential returns the credential support this model was created with.
func (m *Model) DescribeCredential() *CredentialSupport {
	return proto.Clone(m.support).(*CredentialSupport)
}

// GetCredential returns the credential with the given id.
func (m *Model) GetCredential(id string) (*Credential, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.byID[id]
	if !ok {
		return nil, status.Error(codes.NotFound, "credential not found")
	}
	return proto.Clone(c).(*Credential), nil
}

// ListCredentials returns all credentials, in no particular order.
func (m *Model) ListCredentials() []*Credential {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]*Credential, 0, len(m.byID))
	for _, c := range m.byID {
		res = append(res, proto.Clone(c).(*Credential))
	}
	return res
}

// CreateCredential stores a new credential, assigning it an id.
// The id, kind, native_state and invitation.status of c are ignored.
func (m *Model) CreateCredential(c *Credential) (*Credential, error) {
	if c == nil {
		return nil, status.Error(codes.InvalidArgument, "credential is required")
	}
	c = proto.Clone(c).(*Credential)
	c.Id = ""
	c.NativeState = ""
	if c.State == Credential_STATE_UNSPECIFIED {
		c.State = Credential_ACTIVE
	}
	if c.Invitation != nil {
		c.Invitation.Status = ""
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	t, err := m.checkType(c)
	if err != nil {
		return nil, err
	}
	switch {
	case t.ValueSource == CredentialType_SYSTEM_ALLOCATED && c.Value != "":
		return nil, status.Errorf(codes.InvalidArgument, "credential type %q does not accept a value", t.Id)
	case t.ValueSource != CredentialType_CALLER_SUPPLIED && c.Value == "":
		c.Value = m.allocateValue()
	}
	if err := checkState(c, t); err != nil {
		return nil, err
	}
	if err := m.checkStored(c, t); err != nil {
		return nil, err
	}

	m.lastID++
	c.Id = strconv.Itoa(m.lastID)
	m.byID[c.Id] = c
	return proto.Clone(c).(*Credential), nil
}

// UpdateCredential updates the credential identified by c.id, applying only the fields in updateMask.
// A nil updateMask updates all writable fields.
// Output only fields are ignored, and an unset state leaves the state unchanged.
func (m *Model) UpdateCredential(c *Credential, updateMask *fieldmaskpb.FieldMask) (*Credential, error) {
	if c.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "credential.id is required")
	}
	updater := masks.NewFieldUpdater(
		masks.WithWritableFields(m.support.GetResourceSupport().GetWritableFields()),
		masks.WithUpdateMask(updateMask),
	)
	if err := updater.Validate(c); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.byID[c.Id]
	if !ok {
		return nil, status.Error(codes.NotFound, "credential not found")
	}
	updated := proto.Clone(old).(*Credential)
	updater.Merge(updated, proto.Clone(c))
	// never let a merge change output only fields
	updated.Id = old.Id
	updated.NativeState = old.NativeState
	if updated.Invitation != nil {
		updated.Invitation.Status = old.Invitation.GetStatus()
	}
	if updated.State == Credential_STATE_UNSPECIFIED {
		// treating unset as ACTIVE, as create does, would re-activate a lost card whenever a client left state out
		updated.State = old.State
	}

	t, err := m.checkType(updated)
	if err != nil {
		return nil, err
	}
	// the system allocates values for this type, so they can't be edited or carried over from another type
	if t.ValueSource == CredentialType_SYSTEM_ALLOCATED && updated.Value != "" &&
		(updated.Value != old.Value || updated.Type != old.Type) {
		return nil, status.Errorf(codes.InvalidArgument, "credential type %q does not accept a value", t.Id)
	}
	if updated.State != old.State || updated.Type != old.Type {
		if err := checkState(updated, t); err != nil {
			return nil, err
		}
	}
	if err := m.checkStored(updated, t); err != nil {
		return nil, err
	}

	m.byID[updated.Id] = updated
	return proto.Clone(updated).(*Credential), nil
}

// DeleteCredential removes the credential with the given id.
// If allowMissing is false, deleting a credential that doesn't exist is a NotFound error.
func (m *Model) DeleteCredential(id string, allowMissing bool) error {
	if id == "" {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[id]; !ok {
		if allowMissing {
			return nil
		}
		return status.Error(codes.NotFound, "credential not found")
	}
	delete(m.byID, id)
	return nil
}

// checkType finds the CredentialType for c.type and sets c.kind to match.
func (m *Model) checkType(c *Credential) (*CredentialType, error) {
	if c.Type == "" {
		return nil, status.Error(codes.InvalidArgument, "credential.type is required")
	}
	i := slices.IndexFunc(m.support.GetTypes(), func(t *CredentialType) bool { return t.GetId() == c.Type })
	if i < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "unknown credential type %q", c.Type)
	}
	t := m.support.GetTypes()[i]
	c.Kind = t.GetKind()
	return t, nil
}

// checkStored checks the rules every stored credential follows, however it was created or last updated.
func (m *Model) checkStored(c *Credential, t *CredentialType) error {
	if c.Value == "" {
		return status.Error(codes.InvalidArgument, "credential.value is required")
	}
	if t.InvitationRequired && c.Invitation.GetEmail() == "" && c.Invitation.GetPhoneNumber() == "" {
		return status.Errorf(codes.InvalidArgument, "credential type %q requires an invitation", t.Id)
	}
	if c.ActiveTime != nil && c.ExpireTime != nil && !c.ExpireTime.AsTime().After(c.ActiveTime.AsTime()) {
		return status.Error(codes.InvalidArgument, "credential.expire_time must be after active_time")
	}
	return m.checkDuplicate(c)
}

func (m *Model) checkDuplicate(c *Credential) error {
	for _, other := range m.byID {
		if other.Id != c.Id && other.Type == c.Type && other.Value == c.Value {
			return status.Error(codes.AlreadyExists, "a credential with this type and value already exists")
		}
	}
	return nil
}

func (m *Model) allocateValue() string {
next:
	for {
		m.lastValue++
		v := fmt.Sprintf("%08d", m.lastValue)
		for _, c := range m.byID {
			if c.Value == v {
				continue next
			}
		}
		return v
	}
}

func checkState(c *Credential, t *CredentialType) error {
	if ws := t.GetWritableStates(); len(ws) > 0 && !slices.Contains(ws, c.State) {
		return status.Errorf(codes.InvalidArgument, "credential type %q cannot be set to state %v", t.Id, c.State)
	}
	return nil
}
