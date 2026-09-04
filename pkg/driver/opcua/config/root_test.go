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
