package accesstoken

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"go.uber.org/multierr"

	"github.com/smart-core-os/sc-bos/internal/auth/permission"
	"github.com/smart-core-os/sc-bos/internal/util/pass"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/proto/accountpb"
)

// Verifier verifies that an id is associated with a given secret.
type Verifier interface {
	// Verify will check that the provided id+secret pair is correct, and return SecretData describing the
	// authenticated identity if so.
	// Implementations are encouraged to return one of the sentinel errors ErrInvalidCredentials or ErrNoRolesAssigned
	// so that appropriate error codes can be returned to the client. Otherwise, the client will be unable to
	// determine the reason for the failure.
	Verify(ctx context.Context, id, secret string) (SecretData, error)
}

var (
	ErrInvalidCredentials = tokenError{
		Code:             http.StatusBadRequest,
		ErrorName:        "invalid_grant",
		ErrorDescription: "provided credentials are incorrect",
	}
	ErrNoRolesAssigned = tokenError{
		Code:             http.StatusBadRequest,
		ErrorName:        "unauthorized_client",
		ErrorDescription: "no roles assigned that allow access to this resource",
	}
)

// VerifierFunc adapts an ordinary func to implement Verifier.
type VerifierFunc func(ctx context.Context, id, secret string) (SecretData, error)

func (v VerifierFunc) Verify(ctx context.Context, id, secret string) (SecretData, error) {
	return v(ctx, id, secret)
}

// NeverVerify returns a Verifier that always returns the given error.
func NeverVerify(err error) Verifier {
	return VerifierFunc(func(_ context.Context, _, _ string) (SecretData, error) {
		return SecretData{}, err
	})
}

// FirstSuccessfulVerifier implements Verifier returning the first successful response from member Verifiers.
// Each Verifier will be invoked in separate go routines in parallel.
// The first Verifier to return a non-error will attempt to cancel the remaining Verifier.Verify invocations.
// If all members return errors then these will be combined and returned from this call.
type FirstSuccessfulVerifier []Verifier

func (v FirstSuccessfulVerifier) Verify(ctx context.Context, id, secret string) (data SecretData, err error) {
	if len(v) == 1 {
		return v[0].Verify(ctx, id, secret)
	}

	success := make(chan SecretData, 1)
	errs := make([]error, len(v))

	ctx, cancel := context.WithCancel(ctx)
	// Technically we only need to cancel on success as returning due to error means all members have completed and the context is no longer in use.
	// But we do it on any return anyway as it doesn't really hurt anything and removes lint warnings :/
	defer cancel()

	allDone := make(chan struct{})
	var outstandingTasks sync.WaitGroup
	outstandingTasks.Add(len(v))

	for i, source := range v {
		i, source := i, source
		go func() {
			defer outstandingTasks.Done()
			data, err := source.Verify(ctx, id, secret)
			if err != nil {
				errs[i] = err
				return
			}

			// We do this in a select with default in case more than one source succeeds.
			// The receiver on success only accepts 1 response and we don't want the go routine to block.
			select {
			case success <- data:
			default:
			}
		}()
	}

	go func() {
		outstandingTasks.Wait()
		close(allDone)
	}()

	select {
	case data := <-success:
		return data, nil
	case <-allDone:
		// We select on success again here in case there was a race between all tasks completing and one of them succeeding.
		// This can happen depending on the order go routines are woken up.
		// If all tasks complete, at least one of which succeeded, before this go routine is woken
		// then there's a race between this go routine and the outstandingTasks go routine above. If the later is
		// woken first then both the success chan and allDone chan are 'active' and there's no way to know
		// which case will be triggered in that situation. So we check them both.
		select {
		case data := <-success:
			return data, nil
		default:
		}

		return SecretData{}, multierr.Combine(errs...)
	}
}

// MemoryVerifier implements a primitive, in memory store for client secrets.
// A zero MemoryVerifier is ready to use as an empty store. Don't copy once accessed.
// In production, you'd want to store the secrets hashed in a database, so don't use this!
type MemoryVerifier struct {
	m     sync.RWMutex
	store map[string]*memoryRecord // keyed by id
}

type memoryRecord struct {
	m      sync.RWMutex
	data   SecretData
	hashes map[string][]byte // from secret id to secret hash
}

func (m *memoryRecord) genIdLocked() string {
	for {
		id := genId()
		_, exists := m.hashes[id]
		if !exists {
			return id
		}
	}
}

func (m *memoryRecord) genSecretLocked(id string) (string, error) {
	secret := genSecret()
	err := m.saveSecretLocked(id, secret)
	if err != nil {
		return "", err
	}
	return secret, nil
}

func (m *memoryRecord) saveSecretLocked(id, secret string) error {
	// the case where a duplicate hash is generated is vanishingly unlikely, but check for it anyway
	for {
		secretHash, err := pass.Hash([]byte(secret))
		if err != nil {
			return err
		}
		if m.hashExistsLocked(secretHash) {
			continue
		}
		m.hashes[id] = secretHash
		return nil
	}
}

func (m *memoryRecord) hashExistsLocked(hash []byte) bool {
	for _, h := range m.hashes {
		if bytes.Equal(hash, h) {
			return true
		}
	}
	return false
}

func (m *memoryRecord) getSecretId(secret string) (string, bool) {
	for id, hash := range m.hashes {
		if err := pass.Compare(hash, []byte(secret)); err == nil {
			return id, true
		}
	}
	return "", false
}

func (m *memoryRecord) validateLocked(secret string) error {
	_, ok := m.getSecretId(secret)
	if !ok {
		return ErrInvalidCredentials
	}
	if len(m.data.SystemRoles) == 0 && len(m.data.Permissions) == 0 {
		return ErrNoRolesAssigned
	}
	return nil
}

func (m *memoryRecord) replaceLocked(current, replacement string) error {
	id, ok := m.getSecretId(current)
	if !ok {
		return errors.New("no matching secret")
	}
	return m.saveSecretLocked(id, replacement)
}

func (v *MemoryVerifier) Verify(_ context.Context, id, secret string) (SecretData, error) {
	v.m.RLock()
	defer v.m.RUnlock()

	r, ok := v.store[id]
	if !ok {
		return SecretData{}, ErrInvalidCredentials
	}

	r.m.RLock()
	defer r.m.RUnlock()
	err := r.validateLocked(secret)
	if err != nil {
		return SecretData{}, err
	}
	return r.data, nil
}

// AddRecord makes the verifier aware of a new record.
// The record will have no secrets, call MemoryVerifier.CreateSecret to create one.
func (v *MemoryVerifier) AddRecord(data SecretData) error {
	v.m.Lock()
	defer v.m.Unlock()

	v.ensureInitialised()

	_, exists := v.store[data.TenantID]
	if exists {
		return fmt.Errorf("tenant exists '%v'", data.TenantID)
	}

	v.store[data.TenantID] = &memoryRecord{
		data:   data,
		hashes: make(map[string][]byte),
	}
	return nil
}

func (v *MemoryVerifier) CreateSecret(id string) (sId, secret string, err error) {
	v.m.RLock()
	r, ok := v.store[id]
	v.m.RUnlock()
	if !ok {
		return "", "", errors.New("id not recognised")
	}

	r.m.Lock()
	defer r.m.Unlock()
	sId = r.genIdLocked()
	secret, err = r.genSecretLocked(sId)
	return sId, secret, err
}

func (v *MemoryVerifier) AddSecret(id, secret string) (sId string, err error) {
	v.m.RLock()
	r, ok := v.store[id]
	v.m.RUnlock()
	if !ok {
		return "", errors.New("id not recognised")
	}

	r.m.Lock()
	defer r.m.Unlock()
	sId = r.genIdLocked()
	err = r.saveSecretLocked(sId, secret)
	if err != nil {
		return "", err
	}
	return sId, nil
}

func (v *MemoryVerifier) AddSecretHash(id string, hash []byte) (sId string, err error) {
	v.m.RLock()
	r, ok := v.store[id]
	v.m.RUnlock()
	if !ok {
		return "", errors.New("id not recognised")
	}

	r.m.Lock()
	defer r.m.Unlock()
	sId = r.genIdLocked()
	if r.hashExistsLocked(hash) {
		return "", errors.New("hash already exists")
	}
	r.hashes[sId] = hash
	return sId, nil
}

func (v *MemoryVerifier) ReplaceSecret(id, oldSecret string) (secret string, err error) {
	v.m.RLock()
	r, ok := v.store[id]
	v.m.RUnlock()
	if !ok {
		return "", errors.New("id not recognised")
	}

	secret = genSecret()
	err = r.replaceLocked(oldSecret, secret)
	return
}

func (v *MemoryVerifier) UpdateSecret(id, current, replacement string) error {
	v.m.RLock()
	r, ok := v.store[id]
	v.m.RUnlock()
	if !ok {
		return errors.New("id not recognised")
	}

	return r.replaceLocked(current, replacement)
}

func (v *MemoryVerifier) DeleteSecret(id, secretId string) bool {
	v.m.RLock()
	r, ok := v.store[id]
	v.m.RUnlock()
	if !ok {
		return false
	}

	r.m.Lock()
	defer r.m.Unlock()
	_, ok = r.hashes[secretId]
	if !ok {
		return false
	}
	delete(r.hashes, secretId)
	return true
}

func (v *MemoryVerifier) DeleteRecord(id string) bool {
	v.m.Lock()
	defer v.m.Unlock()

	_, exists := v.store[id]
	if exists {
		delete(v.store, id)
	}
	return exists
}

func (v *MemoryVerifier) ensureInitialised() {
	if v.store == nil {
		v.store = make(map[string]*memoryRecord)
	}
}

type SecretData struct {
	Title       string
	TenantID    string
	SystemRoles []string
	IsService   bool
	Permissions []token.PermissionAssignment
}

// LegacyZonePermission returns a PermissionAssignment that grants write access to names beginning with the given zone prefix.
// This does not use a ZONE resource type, in order to maintain compatibility.
func LegacyZonePermission(zone string) token.PermissionAssignment {
	return token.PermissionAssignment{
		Permission:   permission.TraitWrite,
		Scoped:       true,
		ResourceType: token.ResourceType(accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX),
		Resource:     zone,
	}
}

func genId() string {
	var bits = make([]byte, 16)
	_, _ = rand.Reader.Read(bits)
	return base64.StdEncoding.EncodeToString(bits)
}

func genSecret() (secret string) {
	// generate 256 bits of cryptographic-strength random data to use as the secret
	raw := make([]byte, 256/8)
	_, err := rand.Read(raw)
	if err != nil {
		// on a sane platform, random numbers should always be available
		panic(err)
	}

	// encode the secret with URL-safe base64 encoding
	var buffer bytes.Buffer
	encoder := base64.NewEncoder(base64.URLEncoding, &buffer)
	_, err = encoder.Write(raw)
	if err != nil {
		// the base64 encoder should work with any data
		panic(err)
	}
	err = encoder.Close()
	if err != nil {
		panic(err)
	}

	return buffer.String()
}
