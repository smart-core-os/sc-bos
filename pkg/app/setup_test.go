package app

import (
	"encoding/json"
	"testing"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
)

func TestHasOtherLogins(t *testing.T) {
	tests := []struct {
		name     string
		authn    string // the systems.authn config, "" for none
		accounts bool   // experimental.accounts
		want     bool
	}{
		{name: "no authn", want: false},
		{name: "local accounts only", authn: `{"user":{"localAccounts":true}}`, accounts: true, want: false},
		{name: "keycloak", authn: `{"user":{"localAccounts":true,"keycloakAccounts":{"url":"https://kc","realm":"r","clientId":"c"}}}`, accounts: true, want: true},
		{name: "users.json used directly", authn: `{"user":{"fileAccounts":true}}`, accounts: true, want: true},
		{name: "users.json imported", authn: `{"user":{"localAccounts":true,"importFileAccounts":true,"fileAccounts":true}}`, accounts: true, want: false},
		// without the account store nothing is imported, so users.json is still used directly
		{name: "import without account store", authn: `{"user":{"localAccounts":true,"importFileAccounts":true,"fileAccounts":true}}`, want: true},
		{name: "authn disabled", authn: `{"disabled":true,"user":{"fileAccounts":true}}`, accounts: true, want: false},
		{name: "service accounts only", authn: `{"system":{"fileAccounts":true,"tenantAccounts":true}}`, accounts: true, want: false},
		{name: "unreadable", authn: `{"user":"nope"}`, accounts: true, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg sysconf.Config
			if tt.authn != "" {
				raw := `{"systems":{"authn":` + tt.authn + `}}`
				if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
					t.Fatal(err)
				}
			}
			if tt.accounts {
				cfg.Experimental = &sysconf.Experimental{Accounts: true}
			}
			if got := hasOtherLogins(cfg, zap.NewNop()); got != tt.want {
				t.Errorf("hasOtherLogins = %v, want %v", got, tt.want)
			}
		})
	}
}
