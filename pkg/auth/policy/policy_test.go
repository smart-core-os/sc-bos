package policy

import (
	"context"
	"embed"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/open-policy-agent/opa/rego"

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
		expectQueries []string
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
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			policy := &mockPolicy{responses: c.responses}
			_, err := Validate(context.Background(), policy, c.attr)
			if !errors.Is(err, c.expectErr) {
				t.Errorf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(c.expectQueries, policy.queries); diff != "" {
				t.Errorf("wrong query sequence (-want +got):\n%s", diff)
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
