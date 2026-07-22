package pgxutil

import (
	"context"
	"strings"
	"testing"
)

func TestRoleConfig_IsZero(t *testing.T) {
	base := ConnectConfig{URI: "postgres://base"}
	tests := []struct {
		name string
		rc   RoleConfig
		want bool
	}{
		{"empty", RoleConfig{}, true},
		{"base uri", RoleConfig{ConnectConfig: base}, false},
		{"read only", RoleConfig{Read: &base}, false},
		{"write only", RoleConfig{Write: &base}, false},
		{"admin only", RoleConfig{Admin: &base}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rc.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnectRoles_partialOverrideNoBase(t *testing.T) {
	// Only "read" set, no base uri: write and admin resolve to a zero config.
	// ConnectRoles must reject this rather than let the empty configs fall through
	// to libpq env defaults. The error should name the first unconfigured role and
	// no connection should be attempted (the read uri is unreachable in tests).
	rc := RoleConfig{Read: &ConnectConfig{URI: "postgres://read"}}
	_, err := ConnectRoles(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error for partial role override with no base, got nil")
	}
	if !strings.Contains(err.Error(), "write") {
		t.Errorf("error should name the unconfigured write role, got: %v", err)
	}
}

func TestRoleConfig_resolve(t *testing.T) {
	base := ConnectConfig{URI: "postgres://base"}
	r := ConnectConfig{URI: "postgres://read"}
	w := ConnectConfig{URI: "postgres://write"}
	a := ConnectConfig{URI: "postgres://admin"}

	tests := []struct {
		name                    string
		rc                      RoleConfig
		wantR, wantW, wantAdmin ConnectConfig
	}{
		{
			name:  "base only falls back for all roles",
			rc:    RoleConfig{ConnectConfig: base},
			wantR: base, wantW: base, wantAdmin: base,
		},
		{
			name:  "read override, others fall back to base",
			rc:    RoleConfig{ConnectConfig: base, Read: &r},
			wantR: r, wantW: base, wantAdmin: base,
		},
		{
			name:  "each role overridden",
			rc:    RoleConfig{ConnectConfig: base, Read: &r, Write: &w, Admin: &a},
			wantR: r, wantW: w, wantAdmin: a,
		},
		{
			name:  "no base, only role overrides",
			rc:    RoleConfig{Read: &r, Write: &w, Admin: &a},
			wantR: r, wantW: w, wantAdmin: a,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotR, gotW, gotAdmin := tt.rc.resolve()
			if gotR != tt.wantR {
				t.Errorf("read = %v, want %v", gotR, tt.wantR)
			}
			if gotW != tt.wantW {
				t.Errorf("write = %v, want %v", gotW, tt.wantW)
			}
			if gotAdmin != tt.wantAdmin {
				t.Errorf("admin = %v, want %v", gotAdmin, tt.wantAdmin)
			}
		})
	}
}
