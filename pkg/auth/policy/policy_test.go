package policy

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/smart-core-os/sc-bos/internal/auth/permission"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/proto/accountpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
)

func TestValidate(t *testing.T) {
	allow := rego.ResultSet{{
		Expressions: []*rego.ExpressionValue{
			{
				Value: true,
				Text:  "allow",
			},
		},
	}}

	deny := rego.ResultSet{{
		Expressions: []*rego.ExpressionValue{
			{
				Value: false,
				Text:  "allow",
			},
		},
	}}

	empty := rego.ResultSet{}

	type testCase struct {
		attr          Attributes
		responses     map[string]rego.ResultSet
		expectErr     error
		expectQueries []string // queries called on the policy
		expectTried   []string // queries returned in the tried slice
	}

	cases := map[string]testCase{
		"Hierarchy": {
			attr: Attributes{
				Protocol: ProtocolGRPC,
				Service:  "foo.bar.baz",
			},
			expectErr: ErrUnauthenticated,
			expectQueries: []string{
				"data.foo.bar.baz.allow",
				"data.foo.bar.allow",
				"data.foo.allow",
				"data.grpc_default.allow",
			},
			expectTried: []string{
				"data.foo.bar.baz.allow",
				"data.foo.bar.allow",
				"data.foo.allow",
				"data.grpc_default.allow",
			},
		},
		"ShortCircuit_Positive": {
			attr: Attributes{
				Protocol: ProtocolGRPC,
				Service:  "foo.bar.baz",
			},
			responses: map[string]rego.ResultSet{
				"data.foo.bar.baz.allow": empty,
				"data.foo.bar.allow":     allow,
				"data.foo.allow":         deny,
			},
			expectErr: nil,
			expectQueries: []string{
				"data.foo.bar.baz.allow",
				"data.foo.bar.allow",
			},
			expectTried: []string{
				"data.foo.bar.baz.allow",
				"data.foo.bar.allow",
			},
		},
		"ShortCircuit_Negative": {
			attr: Attributes{
				Protocol: ProtocolGRPC,
				Service:  "foo.bar.baz",
			},
			responses: map[string]rego.ResultSet{
				"data.foo.bar.baz.allow": empty,
				"data.foo.bar.allow":     deny,
				"data.foo.allow":         allow,
			},
			expectErr: ErrUnauthenticated,
			expectQueries: []string{
				"data.foo.bar.baz.allow",
				"data.foo.bar.allow",
			},
			expectTried: []string{
				"data.foo.bar.baz.allow",
				"data.foo.bar.allow",
			},
		},
		"PermissionDenied_token": {
			attr: Attributes{
				Protocol:     ProtocolGRPC,
				Service:      "foo.bar.baz",
				TokenPresent: true,
				TokenValid:   true,
			},
			responses: map[string]rego.ResultSet{
				"data.foo.bar.baz.allow": deny,
			},
			expectErr: ErrPermissionDenied,
			expectQueries: []string{
				"data.foo.bar.baz.allow",
			},
			expectTried: []string{
				"data.foo.bar.baz.allow",
			},
		},
		"PermissionDenied_cert": {
			attr: Attributes{
				Protocol:           ProtocolGRPC,
				Service:            "foo.bar.baz",
				CertificatePresent: true,
				CertificateValid:   true,
			},
			responses: map[string]rego.ResultSet{
				"data.foo.bar.baz.allow": deny,
			},
			expectErr: ErrPermissionDenied,
			expectQueries: []string{
				"data.foo.bar.baz.allow",
			},
			expectTried: []string{
				"data.foo.bar.baz.allow",
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			policy := &mockPolicy{responses: c.responses}
			tried, err := Validate(context.Background(), policy, c.attr)
			if !errors.Is(err, c.expectErr) {
				t.Errorf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(c.expectQueries, policy.queries); diff != "" {
				t.Errorf("wrong query sequence (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(c.expectTried, tried); diff != "" {
				t.Errorf("wrong tried slice (-want +got):\n%s", diff)
			}
		})
	}
}

type mockPolicy struct {
	responses map[string]rego.ResultSet
	queries   []string
}

func (p *mockPolicy) EvalPolicy(ctx context.Context, query string, attributes Attributes) (rego.ResultSet, error) {
	p.queries = append(p.queries, query)
	return p.responses[query], nil
}

//go:embed testdata
var testdata embed.FS

func TestValidate_Integration(t *testing.T) {
	policy, err := FromFS(testdata)
	if err != nil {
		t.Fatal(err)
	}

	type testCase struct {
		attr      Attributes
		expectErr error
	}
	cases := map[string]testCase{
		"foo.bar": {
			attr: Attributes{
				Protocol: ProtocolGRPC,
				Service:  "foo.bar",
			},
			expectErr: ErrUnauthenticated,
		},
		"foo.baz": {
			attr: Attributes{
				Protocol: ProtocolGRPC,
				Service:  "foo.baz",
			},
			expectErr: nil,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Validate(context.Background(), policy, c.attr)
			if !errors.Is(err, c.expectErr) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// tests that the policy knows which BOS APIs are traits
func TestDefaultPolicy_Traits(t *testing.T) {
	policy := Default(false)

	attrs := Attributes{
		Protocol: ProtocolGRPC,
		Service:  "smartcore.bos.soundsensor.v1.SoundSensorApi",
		Method:   "GetSoundLevel",
		Request: &soundsensorpb.GetSoundLevelRequest{
			Name: "foo/testsoundsensor",
		},
		TokenPresent: true,
		TokenValid:   true,
		TokenClaims: token.Claims{
			Permissions: []token.PermissionAssignment{
				{
					Permission:   permission.TraitRead,
					Scoped:       true,
					ResourceType: token.ResourceType(accountpb.RoleAssignment_NAMED_RESOURCE_PATH_PREFIX),
					Resource:     "foo",
				},
			},
		},
	}
	_, err := Validate(context.Background(), policy, attrs)
	if err != nil {
		t.Errorf("expected access to be allowed, got error: %v", err)
	}

	// try an API that has nothing to do with any known trait
	attrs.Service = "smartcore.bos.NonExistentTraitApi"
	_, err = Validate(context.Background(), policy, attrs)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("%s: expected permission denied, got: %v", attrs.Service, err)
	}

	// try an API that looks like a trait API, but isn't registered
	// this would have been allowed by a previous, looser implementation of trait matching, but shouldn't be
	attrs.Service = "smartcore.bos.SoundSensorFoobar"
	_, err = Validate(context.Background(), policy, attrs)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("%s: expected permission denied, got: %v", attrs.Service, err)
	}
}

// benchmarkScenarios is a mix of realistic server requests that exercise different
// hierarchy depths and auth contexts, representative of a loaded BOS server:
//
//   - 1-query path:  operator + TenantApi write verb (stops at the specific-package rule)
//   - 3-query path:  token/cert + trait API (falls through to data.smartcore.allow)
//   - 5-query path:  TenantApi reads/writes by various roles (walks all BOS layers to data.smartcore.allow)
//   - full-walk:     unauthenticated request (all queries empty, walks the complete hierarchy)
var benchmarkScenarios = []Attributes{
	// 1-query stop: operator + TenantApi verb that matches the specific-package rule
	{
		Protocol: ProtocolGRPC, Service: "smartcore.bos.tenants.v1.TenantApi", Method: "AddTenant",
		Request: json.RawMessage(`{"name":"test"}`), TokenPresent: true, TokenValid: true,
		TokenClaims: token.Claims{SystemRoles: []string{"operator"}},
	},
	// 3-query stop: admin reads a trait (falls through traits package, stops at smartcore.allow)
	{
		Protocol: ProtocolGRPC, Service: "smartcore.traits.OnOff", Method: "GetOnOff",
		Request: json.RawMessage(`{"name":"devices/light1"}`), TokenPresent: true, TokenValid: true,
		TokenClaims: token.Claims{SystemRoles: []string{"admin"}},
	},
	// 3-query stop: operator writes a trait
	{
		Protocol: ProtocolGRPC, Service: "smartcore.traits.OnOff", Method: "UpdateOnOff",
		Request: json.RawMessage(`{"name":"devices/light1","onOff":{"state":"ON"}}`), TokenPresent: true, TokenValid: true,
		TokenClaims: token.Claims{SystemRoles: []string{"operator"}},
	},
	// 3-query stop: viewer reads a trait
	{
		Protocol: ProtocolGRPC, Service: "smartcore.traits.Brightness", Method: "GetBrightness",
		Request: json.RawMessage(`{"name":"devices/light2"}`), TokenPresent: true, TokenValid: true,
		TokenClaims: token.Claims{SystemRoles: []string{"viewer"}},
	},
	// 3-query stop: certificate auth reads a trait (cert rule fires at smartcore.allow)
	{
		Protocol: ProtocolGRPC, Service: "smartcore.traits.OnOff", Method: "GetOnOff",
		Request:            json.RawMessage(`{"name":"devices/light3"}`),
		CertificatePresent: true, CertificateValid: true,
	},
	// 5-query stop: operator reads from TenantApi (verb doesn't match specific-package rule;
	// falls all the way through BOS layers to smartcore.allow via operator+read)
	{
		Protocol: ProtocolGRPC, Service: "smartcore.bos.tenants.v1.TenantApi", Method: "GetTenant",
		Request: json.RawMessage(`{"name":"test"}`), TokenPresent: true, TokenValid: true,
		TokenClaims: token.Claims{SystemRoles: []string{"operator"}},
	},
	// 5-query stop: admin deletes from TenantApi (no admin rule in specific package;
	// walks to smartcore.allow where admin is unconditionally allowed)
	{
		Protocol: ProtocolGRPC, Service: "smartcore.bos.tenants.v1.TenantApi", Method: "DeleteCredential",
		Request: json.RawMessage(`{"name":"test"}`), TokenPresent: true, TokenValid: true,
		TokenClaims: token.Claims{SystemRoles: []string{"admin"}},
	},
	// full-walk: unauthenticated — no rule fires anywhere, exhausts the entire hierarchy
	{
		Protocol: ProtocolGRPC, Service: "smartcore.traits.OnOff", Method: "GetOnOff",
		Request: json.RawMessage(`{"name":"devices/light4"}`),
	},
}

func BenchmarkValidate_Concurrent(b *testing.B) {
	const goroutines = 16

	run := func(b *testing.B, policy Policy) {
		var wg sync.WaitGroup
		work := make(chan Attributes)
		for range goroutines {
			wg.Go(func() {
				for attr := range work {
					_, _ = Validate(b.Context(), policy, attr)
				}
			})
		}

		b.ResetTimer()
		for i := range b.N {
			attr := benchmarkScenarios[i%len(benchmarkScenarios)]
			work <- attr
		}
		close(work)
		wg.Wait()
	}

	b.Run("static", func(b *testing.B) {
		run(b, Default(false))
	})
	b.Run("cachedStatic", func(b *testing.B) {
		run(b, Default(true))
	})
}
