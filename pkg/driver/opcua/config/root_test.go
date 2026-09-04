package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func TestParseConfig_security(t *testing.T) {
	passwordFile := writePasswordFile(t, "secret")
	certFile, keyFile := quote("/etc/sc-bos/opcua/client.pem"), quote("/etc/sc-bos/opcua/client.key")

	tests := []struct {
		name string
		conn string
		want ResolvedSecurity
		// wantErr, when set, is a substring the parse error must contain.
		wantErr string
	}{
		{
			name: "no auth or security",
			conn: `{"endpoint": "opc.tcp://server:4840"}`,
			want: ResolvedSecurity{
				PolicyURI: ua.SecurityPolicyURINone,
				Mode:      ua.MessageSecurityModeNone,
				TokenType: ua.UserTokenTypeAnonymous,
			},
		},
		{
			name: "auth defaults to an encrypted channel",
			conn: `{"endpoint": "opc.tcp://server:4840", "auth": {"username": "sc-bos", "passwordFile": ` + quote(passwordFile) + `},
				"security": {"certFile": ` + certFile + `, "keyFile": ` + keyFile + `}}`,
			want: ResolvedSecurity{
				PolicyURI: ua.SecurityPolicyURIBasic256Sha256,
				Mode:      ua.MessageSecurityModeSignAndEncrypt,
				TokenType: ua.UserTokenTypeUserName,
				CertFile:  "/etc/sc-bos/opcua/client.pem",
				KeyFile:   "/etc/sc-bos/opcua/client.key",
			},
		},
		{
			name: "auth with no security block at all needs a key pair",
			conn: `{"endpoint": "opc.tcp://server:4840", "auth": {"username": "sc-bos", "passwordFile": ` + quote(passwordFile) + `}}`,
			// the default mode is SignAndEncrypt, which can't be set up without a key pair,
			// so this reports the missing files rather than silently downgrading
			wantErr: "certFile and keyFile are required",
		},
		{
			name: "auth over an explicitly unsecured channel",
			conn: `{"endpoint": "opc.tcp://server:4840", "auth": {"username": "sc-bos", "passwordFile": ` + quote(passwordFile) + `},
				"security": {"policy": "None", "mode": "None"}}`,
			want: ResolvedSecurity{
				PolicyURI: ua.SecurityPolicyURINone,
				Mode:      ua.MessageSecurityModeNone,
				TokenType: ua.UserTokenTypeUserName,
			},
		},
		{
			name: "security without auth stays anonymous",
			conn: `{"endpoint": "opc.tcp://server:4840",
				"security": {"policy": "Basic256", "mode": "Sign", "certFile": ` + certFile + `, "keyFile": ` + keyFile + `}}`,
			want: ResolvedSecurity{
				PolicyURI: ua.SecurityPolicyURIBasic256,
				Mode:      ua.MessageSecurityModeSign,
				TokenType: ua.UserTokenTypeAnonymous,
				CertFile:  "/etc/sc-bos/opcua/client.pem",
				KeyFile:   "/etc/sc-bos/opcua/client.key",
			},
		},
		{
			name:    "plaintext password rejected",
			conn:    `{"endpoint": "opc.tcp://server:4840", "auth": {"username": "sc-bos", "password": "hunter2"}}`,
			wantErr: "plaintext passwords in config are not supported",
		},
		{
			name:    "username without a password file rejected",
			conn:    `{"endpoint": "opc.tcp://server:4840", "auth": {"username": "sc-bos"}}`,
			wantErr: "passwordFile is required",
		},
		{
			name:    "password file without a username rejected",
			conn:    `{"endpoint": "opc.tcp://server:4840", "auth": {"passwordFile": ` + quote(passwordFile) + `}}`,
			wantErr: "username is required",
		},
		{
			name:    "missing key pair under SignAndEncrypt rejected",
			conn:    `{"endpoint": "opc.tcp://server:4840", "security": {"policy": "Basic256Sha256", "mode": "SignAndEncrypt", "certFile": ` + certFile + `}}`,
			wantErr: "certFile and keyFile are required",
		},
		{
			name:    "unknown policy rejected",
			conn:    `{"endpoint": "opc.tcp://server:4840", "security": {"policy": "Basic256Sha257", "mode": "None"}}`,
			wantErr: `unknown policy "Basic256Sha257"`,
		},
		{
			name:    "unknown mode rejected",
			conn:    `{"endpoint": "opc.tcp://server:4840", "security": {"policy": "None", "mode": "SignAndEncypt"}}`,
			wantErr: `unknown mode "SignAndEncypt"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := `{"name": "opcua", "type": "opcua", "conn": ` + tt.conn + `}`
			cfg, err := ParseConfig([]byte(data))
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("ParseConfig succeeded, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ParseConfig error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseConfig: %v", err)
			}
			got, err := cfg.Conn.ResolveSecurity()
			if err != nil {
				t.Fatalf("ResolveSecurity: %v", err)
			}
			if got != tt.want {
				t.Errorf("ResolveSecurity() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolvedSecurity_AnonymousInsecure(t *testing.T) {
	tests := []struct {
		name string
		conn Conn
		want bool
	}{
		{name: "no auth or security", conn: Conn{}, want: true},
		{
			name: "explicit None",
			conn: Conn{Security: &Security{Policy: "None", Mode: "None"}},
			want: true,
		},
		{
			name: "secure channel",
			conn: Conn{Security: &Security{Policy: "Basic256Sha256", Mode: "SignAndEncrypt", CertFile: "cert.pem", KeyFile: "key.pem"}},
			want: false,
		},
		{
			name: "credentials over an unsecured channel",
			conn: Conn{
				Auth:     &Auth{Username: "sc-bos", Password: jsontypes.Password{PasswordFile: "/run/secrets/opcua"}},
				Security: &Security{Policy: "None", Mode: "None"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec, err := tt.conn.ResolveSecurity()
			if err != nil {
				t.Fatalf("ResolveSecurity: %v", err)
			}
			if got := sec.AnonymousInsecure(); got != tt.want {
				t.Errorf("AnonymousInsecure() = %v, want %v", got, tt.want)
			}
		})
	}
}

// writePasswordFile writes contents to a file in a temp dir and returns its path.
func writePasswordFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte(contents+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// quote renders path as a JSON string literal, which escapes the backslashes in Windows paths.
func quote(path string) string {
	return strconv.Quote(path)
}

// TestParseConfig_monitoringDefaults checks the monitoring parameters we send to the server
// when a config leaves them out. Sampling once per publish into a queue of one is what keeps
// the server from overflowing its queue and flagging every value it sends us.
func TestParseConfig_monitoringDefaults(t *testing.T) {
	tests := []struct {
		name                           string
		conn                           string
		wantSubscription, wantSampling time.Duration
		wantQueueSize                  uint32
	}{
		{
			name:             "all defaulted",
			conn:             `{"endpoint": "opc.tcp://server:4840"}`,
			wantSubscription: 5 * time.Second,
			wantSampling:     5 * time.Second,
			wantQueueSize:    1,
		},
		{
			name:             "sampling follows subscription interval",
			conn:             `{"endpoint": "opc.tcp://server:4840", "subscriptionInterval": "1s"}`,
			wantSubscription: time.Second,
			wantSampling:     time.Second,
			wantQueueSize:    1,
		},
		{
			name:             "explicit values are kept",
			conn:             `{"endpoint": "opc.tcp://server:4840", "subscriptionInterval": "5s", "samplingInterval": "250ms", "queueSize": 20}`,
			wantSubscription: 5 * time.Second,
			wantSampling:     250 * time.Millisecond,
			wantQueueSize:    20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConfig([]byte(`{"name": "opcua", "type": "opcua", "conn": ` + tt.conn + `}`))
			if err != nil {
				t.Fatalf("ParseConfig: %v", err)
			}
			if got := cfg.Conn.SubscriptionInterval.Duration; got != tt.wantSubscription {
				t.Errorf("subscriptionInterval = %v, want %v", got, tt.wantSubscription)
			}
			if got := cfg.Conn.SamplingInterval.Duration; got != tt.wantSampling {
				t.Errorf("samplingInterval = %v, want %v", got, tt.wantSampling)
			}
			if got := cfg.Conn.QueueSize; got != tt.wantQueueSize {
				t.Errorf("queueSize = %d, want %d", got, tt.wantQueueSize)
			}
		})
	}
}

// TestParseConfig_monitoringRejected checks the intervals a server cannot act on sensibly are
// refused at parse time. "0s" is the one that matters: OPC UA reads a sampling interval of 0 as
// "as fast as you can", so it used to be a silent opt-in to the fastest rate the server offered.
func TestParseConfig_monitoringRejected(t *testing.T) {
	tests := []struct {
		name    string
		conn    string
		wantErr string
	}{
		{
			name:    "zero sampling interval",
			conn:    `{"endpoint": "opc.tcp://server:4840", "samplingInterval": "0s"}`,
			wantErr: "samplingInterval must be positive",
		},
		{
			name:    "negative sampling interval",
			conn:    `{"endpoint": "opc.tcp://server:4840", "samplingInterval": "-1s"}`,
			wantErr: "samplingInterval must be positive",
		},
		{
			name:    "zero subscription interval",
			conn:    `{"endpoint": "opc.tcp://server:4840", "subscriptionInterval": "0s"}`,
			wantErr: "subscriptionInterval must be positive",
		},
		{
			name:    "negative subscription interval",
			conn:    `{"endpoint": "opc.tcp://server:4840", "subscriptionInterval": "-500ms"}`,
			wantErr: "subscriptionInterval must be positive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConfig([]byte(`{"name": "opcua", "type": "opcua", "conn": ` + tt.conn + `}`))
			if err == nil {
				t.Fatalf("ParseConfig() error = nil, want one containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ParseConfig() error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestConn_MonitoringWarnings(t *testing.T) {
	tests := []struct {
		name string
		conn string
		// want is a substring each expected warning must contain, one per warning, in order.
		want []string
	}{
		{
			name: "defaults warn about nothing",
			conn: `{"endpoint": "opc.tcp://server:4840"}`,
		},
		{
			name: "sampling slower than publishing warns about nothing",
			conn: `{"endpoint": "opc.tcp://server:4840", "subscriptionInterval": "1s", "samplingInterval": "5s"}`,
		},
		{
			name: "queue sized to match the sampling rate warns about nothing",
			conn: `{"endpoint": "opc.tcp://server:4840", "subscriptionInterval": "5s", "samplingInterval": "1s", "queueSize": 5}`,
		},
		{
			name: "queue too small for the sampling rate",
			// the reported fault: 5s cycles sampled every 250ms into the old 10-deep queue
			conn: `{"endpoint": "opc.tcp://server:4840", "subscriptionInterval": "5s", "samplingInterval": "250ms", "queueSize": 10}`,
			want: []string{"raise queueSize to at least 20"},
		},
		{
			name: "queue one short still warns",
			conn: `{"endpoint": "opc.tcp://server:4840", "subscriptionInterval": "5s", "samplingInterval": "1s", "queueSize": 4}`,
			want: []string{"raise queueSize to at least 5"},
		},
		{
			name: "aggressive sampling interval",
			conn: `{"endpoint": "opc.tcp://server:4840", "subscriptionInterval": "5s", "samplingInterval": "50ms", "queueSize": 100}`,
			want: []string{"samplingInterval 50ms is shorter than 100ms"},
		},
		{
			name: "aggressive publishing interval",
			// sampling defaults to the publishing interval, so both are aggressive together
			conn: `{"endpoint": "opc.tcp://server:4840", "subscriptionInterval": "50ms"}`,
			want: []string{"samplingInterval 50ms is shorter than 100ms", "subscriptionInterval 50ms is shorter than 100ms"},
		},
		{
			name: "aggressive on every count",
			conn: `{"endpoint": "opc.tcp://server:4840", "subscriptionInterval": "50ms", "samplingInterval": "10ms"}`,
			want: []string{"raise queueSize to at least 5", "samplingInterval 10ms", "subscriptionInterval 50ms"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConfig([]byte(`{"name": "opcua", "type": "opcua", "conn": ` + tt.conn + `}`))
			if err != nil {
				t.Fatalf("ParseConfig: %v", err)
			}
			got := cfg.Conn.MonitoringWarnings()
			if len(got) != len(tt.want) {
				t.Fatalf("MonitoringWarnings() returned %d warnings, want %d: %s", len(got), len(tt.want), strings.Join(got, " | "))
			}
			for i, want := range tt.want {
				if !strings.Contains(got[i], want) {
					t.Errorf("warning %d = %q, want it to contain %q", i, got[i], want)
				}
			}
		})
	}
}

func Test_samplesPerCycle(t *testing.T) {
	tests := []struct {
		publish, sample time.Duration
		want            int64
	}{
		{publish: 5 * time.Second, sample: 5 * time.Second, want: 1},
		{publish: 5 * time.Second, sample: 10 * time.Second, want: 1},
		{publish: 5 * time.Second, sample: 250 * time.Millisecond, want: 20},
		{publish: 5 * time.Second, sample: time.Second, want: 5},
		// not a whole number of samples per cycle, so round up
		{publish: 5 * time.Second, sample: 3 * time.Second, want: 2},
		{publish: time.Second, sample: 300 * time.Millisecond, want: 4},
		// a non-positive sample interval never reaches here, but must not divide by zero
		{publish: 5 * time.Second, sample: 0, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.publish.String()+"/"+tt.sample.String(), func(t *testing.T) {
			if got := samplesPerCycle(tt.publish, tt.sample); got != tt.want {
				t.Errorf("samplesPerCycle(%s, %s) = %d, want %d", tt.publish, tt.sample, got, tt.want)
			}
		})
	}
}
