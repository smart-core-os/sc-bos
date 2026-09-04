package config

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/gopcua/opcua/ua"
	"math/rand/v2"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

const (
	PointsEventTopicSuffix = "/event/pointset"
)

// valueSourceField represents a ValueSource field with its description for validation.
type valueSourceField struct {
	desc  string
	value *ValueSource
}

// OccupantImpact wraps healthpb.HealthCheck_OccupantImpact to support JSON unmarshaling from strings.
type OccupantImpact healthpb.HealthCheck_OccupantImpact

func (o *OccupantImpact) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	s = strings.ToUpper(s)
	val, ok := healthpb.HealthCheck_OccupantImpact_value[s]
	if !ok {
		return fmt.Errorf("invalid OccupantImpact value: %q (valid values: OCCUPANT_IMPACT_UNSPECIFIED, NO_OCCUPANT_IMPACT, COMFORT, HEALTH, LIFE, SECURITY)", s)
	}

	*o = OccupantImpact(val)
	return nil
}

func (o OccupantImpact) MarshalJSON() ([]byte, error) {
	name := healthpb.HealthCheck_OccupantImpact_name[int32(o)]
	if name == "" {
		return nil, fmt.Errorf("invalid OccupantImpact value: %d", o)
	}
	return json.Marshal(name)
}

func (o OccupantImpact) ToProto() healthpb.HealthCheck_OccupantImpact {
	return healthpb.HealthCheck_OccupantImpact(o)
}

// EquipmentImpact wraps healthpb.HealthCheck_EquipmentImpact to support JSON unmarshaling from strings.
type EquipmentImpact healthpb.HealthCheck_EquipmentImpact

func (e *EquipmentImpact) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	s = strings.ToUpper(s)
	val, ok := healthpb.HealthCheck_EquipmentImpact_value[s]
	if !ok {
		return fmt.Errorf("invalid EquipmentImpact value: %q (valid values: EQUIPMENT_IMPACT_UNSPECIFIED, NO_EQUIPMENT_IMPACT, WARRANTY, LIFESPAN, FUNCTION)", s)
	}

	*e = EquipmentImpact(val)
	return nil
}

func (e EquipmentImpact) MarshalJSON() ([]byte, error) {
	name := healthpb.HealthCheck_EquipmentImpact_name[int32(e)]
	if name == "" {
		return nil, fmt.Errorf("invalid EquipmentImpact value: %d", e)
	}
	return json.Marshal(name)
}

func (e EquipmentImpact) ToProto() healthpb.HealthCheck_EquipmentImpact {
	return healthpb.HealthCheck_EquipmentImpact(e)
}

// Conn config related to communicating with the OPC UA server.
type Conn struct {
	// Endpoint is the OPC UA server endpoint.
	Endpoint string `json:"endpoint,omitempty"`
	// SubscriptionInterval for OPC UA subscription, defaults to 5s if not set.
	// This is the publishing interval: how often the server sends us the samples it has queued.
	SubscriptionInterval *jsontypes.Duration `json:"subscriptionInterval,omitempty,omitzero"`
	// SamplingInterval is how often the server samples the monitored node.
	// Defaults to SubscriptionInterval. A shorter interval than the publishing
	// interval requires QueueSize to be raised to match or samples are discarded.
	SamplingInterval *jsontypes.Duration `json:"samplingInterval,omitempty,omitzero"`
	// QueueSize is the server-side queue depth per monitored item.
	// Defaults to 1, meaning only the most recent sample is published.
	QueueSize uint32 `json:"queueSize,omitempty,omitzero"`
	// ClientId is the ID of the client that will be used to connect to the OPC UA server.
	// Should be unique within the context of a server. If not set, a random ID will be generated.
	ClientId uint32 `json:"clientId,omitempty,omitzero"`

	// Auth configures the OPC UA user identity token.
	// When absent the driver connects to the server anonymously.
	Auth *Auth `json:"auth,omitempty"`
	// Security configures the secure channel used to talk to the server.
	// The defaults depend on whether Auth is set, see ResolveSecurity.
	Security *Security `json:"security,omitempty"`
}

const (
	// DefaultSubscriptionInterval is the publishing interval used when conn omits one.
	DefaultSubscriptionInterval = 5 * time.Second
	// DefaultQueueSize is the server-side queue depth per monitored item used when conn omits
	// one. A depth of one holds the latest sample only, so the server never has to discard
	// anything and never reports queue overflow.
	DefaultQueueSize = 1
	// aggressiveInterval is the point below which a publishing or sampling interval is
	// reported as a load risk. It is a sanity threshold for configs written without checking
	// the server: the authoritative floor is the server's own
	// Server/ServerCapabilities/MinSupportedSampleRate, and a server is free to revise
	// anything we ask for, which Client.Subscribe warns about separately.
	aggressiveInterval = 100 * time.Millisecond
)

// validateMonitoring rejects monitoring parameters the server cannot act on sensibly.
// A non-positive interval is the case worth failing on: OPC UA reads a sampling interval of 0
// as "sample as fast as you practicably can", so a config asking for "0s" quietly opts into
// the fastest rate the server will run and is the usual way queue overflow starts. Wanting the
// fastest available rate is a legitimate thing to want, but it should be a deliberate choice
// expressed as a real duration rather than something a zero value falls into.
//
// ParseConfig defaults the absent fields before calling this, so both interval pointers are
// set by the time it runs and only an explicit non-positive value reaches it.
func (c Conn) validateMonitoring() error {
	if d := c.SubscriptionInterval.Duration; d <= 0 {
		return fmt.Errorf("subscriptionInterval must be positive, got %s; omit it to publish every %s", d, DefaultSubscriptionInterval)
	}
	if d := c.SamplingInterval.Duration; d <= 0 {
		return fmt.Errorf("samplingInterval must be positive, got %s; omit it to sample once per publishing interval", d)
	}
	return nil
}

// MonitoringWarnings reports monitoring parameters that are workable but likely to cause
// trouble, as messages ready to log. It returns nil when there is nothing to say.
//
// These are warnings rather than errors because whether they are a problem depends on the
// server: a fast local simulator will happily sample every 50ms, while the DA-wrapper servers
// this driver often talks to will not. Callers should log them and carry on.
//
// Like validateMonitoring this expects the defaults to have been applied already.
func (c Conn) MonitoringWarnings() []string {
	var warnings []string
	publish, sample := c.SubscriptionInterval.Duration, c.SamplingInterval.Duration

	// a queue too shallow to hold a publishing cycle's worth of samples overflows every cycle,
	// which the server reports by setting the Overflow info bit on the values it does send
	if perCycle := samplesPerCycle(publish, sample); perCycle > int64(c.QueueSize) {
		warnings = append(warnings, fmt.Sprintf(
			"queueSize %d is too small to hold the ~%d samples a samplingInterval of %s produces per %s publishing cycle: "+
				"the server will discard the excess and flag the values it does send with the Overflow info bit; "+
				"raise queueSize to at least %d, or lengthen samplingInterval to %s",
			c.QueueSize, perCycle, sample, publish, perCycle, publish))
	}
	if sample < aggressiveInterval {
		warnings = append(warnings, fmt.Sprintf(
			"samplingInterval %s is shorter than %s, which asks the server to sample every monitored node "+
				"more than %d times a second; many servers will clamp this to their MinSupportedSampleRate",
			sample, aggressiveInterval, int64(time.Second/aggressiveInterval)))
	}
	if publish < aggressiveInterval {
		warnings = append(warnings, fmt.Sprintf(
			"subscriptionInterval %s is shorter than %s; the driver creates one subscription per monitored variable, "+
				"so a short publishing interval multiplies into a lot of traffic",
			publish, aggressiveInterval))
	}
	return warnings
}

// samplesPerCycle is how many samples a publishing cycle of publish holds when the server
// samples every sample, rounded up. It reports 1 when sampling no faster than publishing,
// since a cycle then carries at most one sample.
func samplesPerCycle(publish, sample time.Duration) int64 {
	if sample <= 0 || sample >= publish {
		return 1
	}
	perCycle := int64(publish / sample)
	if publish%sample != 0 {
		perCycle++
	}
	return perCycle
}

// Auth configures the OPC UA user identity token used when creating a session.
type Auth struct {
	// Username is the OPC UA user to authenticate as.
	Username string `json:"username,omitempty"`
	// Password supplies the password for Username.
	// Only passwordFile is accepted, a plaintext password is rejected by ParseConfig.
	jsontypes.Password
}

// Security configures the OPC UA secure channel.
type Security struct {
	// Policy is the security policy short name, one of the keys of ua.SecurityPolicyURIs,
	// e.g. "None", "Basic256Sha256". Defaults to "Basic256Sha256" when Conn.Auth is set,
	// "None" otherwise.
	Policy string `json:"policy,omitempty"`
	// Mode is the message security mode, one of "None", "Sign" or "SignAndEncrypt".
	// Defaults to "SignAndEncrypt" when Conn.Auth is set, "None" otherwise.
	Mode string `json:"mode,omitempty"`
	// CertFile names the client X509 certificate, required for the Sign and SignAndEncrypt modes.
	// The certificate needs a URI subject alternative name, which the client sends as its
	// application URI and servers check against the session.
	CertFile string `json:"certFile,omitempty"`
	// KeyFile names the RSA private key matching CertFile.
	// Required for the Sign and SignAndEncrypt modes.
	KeyFile string `json:"keyFile,omitempty"`
}

// ResolvedSecurity holds the connection security settings resolved from a Conn,
// in the form the gopcua client options need them.
type ResolvedSecurity struct {
	// PolicyURI is the canonical security policy URI, e.g. ua.SecurityPolicyURIBasic256Sha256.
	PolicyURI string
	// Mode is the message security mode of the secure channel.
	Mode ua.MessageSecurityMode
	// TokenType selects the user identity token used to create the session.
	TokenType ua.UserTokenType
	// CertFile names the client certificate, empty when unset.
	CertFile string
	// KeyFile names the client private key, empty when unset.
	KeyFile string
}

// AnonymousInsecure reports whether r describes an anonymous session over an unsecured
// channel, which is what a Conn with neither Auth nor Security resolves to.
func (r ResolvedSecurity) AnonymousInsecure() bool {
	return r.TokenType == ua.UserTokenTypeAnonymous &&
		r.PolicyURI == ua.SecurityPolicyURINone &&
		r.Mode == ua.MessageSecurityModeNone
}

// securityModes lists the message security modes we accept in config.
// ua.MessageSecurityModeFromString maps anything it doesn't recognise to Invalid rather
// than reporting an error, so we check the configured string against this list ourselves.
var securityModes = []string{"None", "Sign", "SignAndEncrypt"}

// ResolveSecurity converts the Auth and Security config into the settings the OPC UA
// client needs, validating them along the way. ParseConfig calls it so that bad security
// config is reported at parse time rather than on connect.
//
// With neither Auth nor Security set it resolves to an anonymous session over an
// unsecured channel, which is how the driver connected before either was configurable.
// Setting Auth without Security defaults the channel to Basic256Sha256/SignAndEncrypt:
// we never pick an unencrypted channel for a password on the operator's behalf, they have
// to ask for that explicitly.
//
// The password itself is not read here. ParseConfig only validates the shape of the
// config, the password is read from disk on each connect attempt so that a rotated secret
// is picked up without a restart and never sits in the parsed config.
func (c Conn) ResolveSecurity() (ResolvedSecurity, error) {
	res := ResolvedSecurity{
		PolicyURI: ua.SecurityPolicyURINone,
		Mode:      ua.MessageSecurityModeNone,
		TokenType: ua.UserTokenTypeAnonymous,
	}

	if c.Auth != nil {
		// Password.Password is the plaintext key of the embedded jsontypes.Password
		if c.Auth.Password.Password != "" {
			return ResolvedSecurity{}, fmt.Errorf("auth: plaintext passwords in config are not supported, use passwordFile")
		}
		if c.Auth.Username == "" {
			return ResolvedSecurity{}, fmt.Errorf("auth: username is required")
		}
		if c.Auth.PasswordFile == "" {
			return ResolvedSecurity{}, fmt.Errorf("auth: passwordFile is required")
		}
		res.TokenType = ua.UserTokenTypeUserName
		res.PolicyURI = ua.SecurityPolicyURIBasic256Sha256
		res.Mode = ua.MessageSecurityModeSignAndEncrypt
	}

	if c.Security != nil {
		if p := c.Security.Policy; p != "" {
			// ua.FormatSecurityPolicyURI turns an unknown name into a URI by prefixing it,
			// so check the name against the known policies before converting it.
			uri, ok := ua.SecurityPolicyURIs[p]
			if !ok {
				return ResolvedSecurity{}, fmt.Errorf("security: unknown policy %q, want one of %s", p, strings.Join(securityPolicyNames(), ", "))
			}
			res.PolicyURI = uri
		}
		if m := c.Security.Mode; m != "" {
			if !slices.Contains(securityModes, m) {
				return ResolvedSecurity{}, fmt.Errorf("security: unknown mode %q, want one of %s", m, strings.Join(securityModes, ", "))
			}
			res.Mode = ua.MessageSecurityModeFromString(m)
		}
		res.CertFile, res.KeyFile = c.Security.CertFile, c.Security.KeyFile
	}

	// gopcua can't sign or encrypt without a client key pair
	if res.Mode == ua.MessageSecurityModeSign || res.Mode == ua.MessageSecurityModeSignAndEncrypt {
		if res.CertFile == "" || res.KeyFile == "" {
			return ResolvedSecurity{}, fmt.Errorf("security: certFile and keyFile are required for mode %s", res.Mode)
		}
	}

	return res, nil
}

// securityPolicyNames returns the accepted security policy names, sorted so that error
// messages listing them are stable.
func securityPolicyNames() []string {
	return slices.Sorted(maps.Keys(ua.SecurityPolicyURIs))
}

// Variable is an OPC UA VariableNode, which is essentially a data point which we can read/write to (with permission).
type Variable struct {
	// NodeId identifies the VariableNode in the OPC UA server.
	NodeId string `json:"nodeId,omitempty"`
	// ParsedNodeId is the parsed ua.NodeID.
	ParsedNodeId *ua.NodeID
}

// Device represents a smart core device.
type Device struct {
	// Name the Smart Core device name
	Name string `json:"name,omitempty"`
	// Meta the Smart Core device metadata
	Meta *metadatapb.Metadata `json:"meta,omitempty"`
	// Variables a list of OPC variables the device has
	Variables []*Variable `json:"variables,omitempty"`
	// Traits a map Smart Core traits the device implements
	Traits []RawTrait `json:"traits,omitempty"`
	// Health contains settings for an opc ua device health check
	// If not configured, the occupant and equipment impact will default to UNSPECIFIED
	Health Health `json:"health"`
}

type Health struct {
	OccupantImpact  OccupantImpact  `json:"occupantImpact"`
	EquipmentImpact EquipmentImpact `json:"equipmentImpact"`
}

type Root struct {
	driver.BaseConfig

	Meta    *metadatapb.Metadata `json:"meta,omitempty"`
	Conn    Conn                 `json:"conn"`
	Devices []Device             `json:"devices,omitempty"`
}

func ParseConfig(data []byte) (cfg Root, err error) {
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, err
	}

	if cfg.Conn.SubscriptionInterval == nil {
		cfg.Conn.SubscriptionInterval = &jsontypes.Duration{Duration: DefaultSubscriptionInterval}
	}
	if cfg.Conn.SamplingInterval == nil {
		// sampling no faster than the server publishes keeps the queue from overflowing,
		// which the server would report as a Good status with the Overflow info bit set
		cfg.Conn.SamplingInterval = &jsontypes.Duration{Duration: cfg.Conn.SubscriptionInterval.Duration}
	}
	if cfg.Conn.QueueSize == 0 {
		cfg.Conn.QueueSize = DefaultQueueSize
	}
	if cfg.Conn.ClientId == 0 {
		cfg.Conn.ClientId = rand.Uint32()
	}

	// check the monitoring parameters now so that an unworkable interval is reported here
	// rather than becoming a load problem on the server after deployment
	if err := cfg.Conn.validateMonitoring(); err != nil {
		return cfg, fmt.Errorf("conn: %w", err)
	}

	// resolve the security config now so that bad security settings are reported here
	// rather than on the first connection attempt
	if _, err := cfg.Conn.ResolveSecurity(); err != nil {
		return cfg, fmt.Errorf("conn: %w", err)
	}

	for _, d := range cfg.Devices {
		for _, v := range d.Variables {
			nId, err := ua.ParseNodeID(v.NodeId)
			if err != nil {
				return cfg, err
			}
			v.ParsedNodeId = nId
		}

		if err := validateDeviceTraits(&d); err != nil {
			return cfg, err
		}
	}

	return cfg, nil
}

// validateDeviceTraits validates trait configurations and checks that all nodeIds referenced in traits
// exist in the device's variable list.
func validateDeviceTraits(device *Device) error {
	validNodeIds := make(map[string]bool)
	for _, v := range device.Variables {
		validNodeIds[v.NodeId] = true
	}

	for _, t := range device.Traits {
		var valueSources []valueSourceField
		var err error

		switch t.Kind {
		case meterpb.TraitName:
			valueSources, err = getValueSourcesForTrait[*MeterConfig](device.Name, t.Raw)
		case trait.Electric:
			valueSources, err = getValueSourcesForTrait[*ElectricConfig](device.Name, t.Raw)
		case transportpb.TraitName:
			valueSources, err = getValueSourcesForTrait[*TransportConfig](device.Name, t.Raw)
		case udmipb.TraitName:
			valueSources, err = getValueSourcesForTrait[*UdmiConfig](device.Name, t.Raw)
		default:
			return fmt.Errorf("device '%s': unknown trait kind '%s'", device.Name, t.Kind)
		}
		if err != nil {
			return err
		}

		for _, field := range valueSources {
			if field.value != nil && field.value.NodeId != "" && !validNodeIds[field.value.NodeId] {
				return fmt.Errorf("device '%s': %s references nodeId '%s' which is not in device variables list",
					device.Name, field.desc, field.value.NodeId)
			}
		}
	}

	return nil
}

type traitWithValueSources interface {
	Validate() error
	valueSources() []valueSourceField
}

// getValueSourcesForTrait parses a trait config, validates it, and returns all its ValueSource fields.
func getValueSourcesForTrait[T traitWithValueSources](deviceName string, rawTrait json.RawMessage) ([]valueSourceField, error) {
	cfg := new(T)
	if err := json.Unmarshal(rawTrait, cfg); err != nil {
		return nil, fmt.Errorf("device '%s': failed to parse trait: %w", deviceName, err)
	}
	if err := (*cfg).Validate(); err != nil {
		return nil, fmt.Errorf("device '%s': %w", deviceName, err)
	}
	return (*cfg).valueSources(), nil
}
