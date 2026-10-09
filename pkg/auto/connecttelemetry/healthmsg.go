package connecttelemetry

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
)

// healthMessageVersion is the version of the health set and manifest envelopes.
const healthMessageVersion = 1

// Subject kinds a health set declares, telling Connect how to find the set's entity.
const (
	subjectKindDevice  = "device"
	subjectKindService = "service"
	subjectKindNode    = "node"
)

func healthTopic(prefix string) string {
	return strings.TrimRight(prefix, "/") + "/bos/health"
}

func healthManifestTopic(prefix string) string {
	return strings.TrimRight(prefix, "/") + "/bos/health-manifest"
}

// healthSetMessage carries one resource's complete set of health checks.
type healthSetMessage struct {
	Version   int               `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Resource  string            `json:"resource"`
	Subject   healthSubject     `json:"subject"`
	Hash      string            `json:"hash"`
	Checks    []json.RawMessage `json:"checks"` // never nil: an empty set is [], which removes every check
}

type healthSubject struct {
	Kind string `json:"kind"`
}

// healthManifestMessage lists every resource the node holds checks for, with each set's hash.
type healthManifestMessage struct {
	Version   int             `json:"version"`
	Timestamp time.Time       `json:"timestamp"`
	Resources []manifestEntry `json:"resources"` // never nil
}

type manifestEntry struct {
	NameHash string `json:"nameHash"`
	Hash     string `json:"hash"`
}

// checkJSON encodes checks with the proto's field names and enums by name, as Connect expects.
var checkJSON = protojson.MarshalOptions{UseProtoNames: true}

// buildHealthSet encodes the health set message for a resource.
func buildHealthSet(now time.Time, name, kind, hash string, checks []*healthpb.HealthCheck) ([]byte, error) {
	msg := healthSetMessage{
		Version:   healthMessageVersion,
		Timestamp: now.UTC(),
		Resource:  name,
		Subject:   healthSubject{Kind: kind},
		Hash:      hash,
		Checks:    make([]json.RawMessage, 0, len(checks)),
	}
	for _, c := range checks {
		b, err := checkJSON.Marshal(c)
		if err != nil {
			return nil, err
		}
		msg.Checks = append(msg.Checks, b)
	}
	// json.Marshal compacts each RawMessage, undoing protojson's deliberately unstable whitespace.
	return json.Marshal(msg)
}

// buildHealthManifest encodes the manifest message.
func buildHealthManifest(now time.Time, entries []manifestEntry) ([]byte, error) {
	if entries == nil {
		entries = []manifestEntry{}
	}
	return json.Marshal(healthManifestMessage{
		Version:   healthMessageVersion,
		Timestamp: now.UTC(),
		Resources: entries,
	})
}

// checkSetHash returns the hash of a resource's set of checks, as declared by kind.
// Measured values, bounds.current_value and the deviation derived from it, are left out
// so a value moving within the same state is not a change.
// The order of checks does not matter.
//
// Only equality is meaningful to Connect, so the hash can change between versions of this
// code, costing one resend of every set.
func checkSetHash(kind string, checks []*healthpb.HealthCheck) string {
	sorted := slices.Clone(checks)
	slices.SortFunc(sorted, func(a, b *healthpb.HealthCheck) int {
		return strings.Compare(a.GetId(), b.GetId())
	})

	h := sha256.New()
	writeLenPrefixed := func(b []byte) {
		_ = binary.Write(h, binary.BigEndian, uint64(len(b)))
		h.Write(b)
	}
	writeLenPrefixed([]byte(kind))
	marshal := proto.MarshalOptions{Deterministic: true}
	for _, c := range sorted {
		c = proto.Clone(c).(*healthpb.HealthCheck)
		if b := c.GetBounds(); b != nil {
			b.CurrentValue = nil
		}
		c.Deviation = 0
		b, err := marshal.Marshal(c)
		if err != nil {
			// HealthCheck has no required fields or invalid UTF-8 source, so this can't happen.
			// Hash the id alone rather than failing the whole set.
			b = []byte(c.GetId())
		}
		writeLenPrefixed(b)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// nameHash is the first 16 hex digits of the SHA-256 of name.
// Unlike checkSetHash this is part of the contract with Connect, which computes it too.
func nameHash(name string) string {
	sum := sha256.Sum256([]byte(name))
	return hex.EncodeToString(sum[:])[:16]
}

// subjectKind returns the subject kind for a resource announced with deviceType.
func subjectKind(deviceType metadatapb.Metadata_DeviceType) string {
	switch deviceType {
	case metadatapb.Metadata_NODE, metadatapb.Metadata_GATEWAY, metadatapb.Metadata_HUB:
		return subjectKindNode
	case metadatapb.Metadata_SERVICE:
		return subjectKindService
	default:
		return subjectKindDevice
	}
}

// dailySlot returns the minute of the UTC day at which name's set is resent.
// Slots are spread across the day by name, and are stable across restarts.
func dailySlot(name string) time.Duration {
	sum := sha256.Sum256([]byte(name))
	const minutesPerDay = 24 * 60
	return time.Duration(binary.BigEndian.Uint64(sum[:8])%minutesPerDay) * time.Minute
}
