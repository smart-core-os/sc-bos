package policy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/open-policy-agent/opa/rego"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/node/alltraits"
)

func BenchmarkPolicy(b *testing.B) {
	attrs := Attributes{
		Service:    "smartcore.traits.OnOff",
		Method:     "UpdateOnOff",
		Request:    json.RawMessage(`{"name":"test","onOff":{"state":"ON"}}`),
		TokenValid: true,
		TokenClaims: token.Claims{
			SystemRoles: []string{"admin"},
		},
	}
	run := func(b *testing.B, policy Policy) {
		for i := 0; i < b.N; i++ {
			result, err := policy.EvalPolicy(context.Background(), "data.smartcore.allow", attrs)
			if err != nil {
				b.Error(err)
			}

			if !result.Allowed() {
				b.Errorf("expected interation %d to suceed", i)
			}
		}
	}

	b.Run("static", func(b *testing.B) {
		policy := Default(false)
		run(b, policy)
	})

	b.Run("cachedStatic", func(b *testing.B) {
		policy := Default(true)
		run(b, policy)
	})
}

func TestSystemData(t *testing.T) {
	test := func(t *testing.T, policy Policy) {
		result, err := policy.EvalPolicy(context.Background(), "data.system.known_traits", Attributes{})
		if err != nil {
			t.Fatal(err)
		}

		if len(result) == 0 {
			t.Fatal("expected result")
		}

		knownTraitsResult, ok := result[0].Expressions[0].Value.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", result[0].Expressions[0].Value)
		}

		if len(knownTraitsResult) != len(alltraits.Names()) {
			t.Errorf("expected %d known traits, got %d", len(alltraits.Names()), len(knownTraitsResult))
		}
	}

	t.Run("static", func(t *testing.T) {
		policy := Default(false)
		test(t, policy)
	})

	t.Run("cachedStatic", func(t *testing.T) {
		policy := Default(true)
		test(t, policy)
	})
}

// regoConsistencyCase captures one (query, input, role) the test will evaluate
// against both the cached and uncached policy implementations.
type regoConsistencyCase struct {
	name   string
	query  string
	method string
	role   string
}

// regoConsistencyCases exercises the hierarchy queries Validate would issue
// for a TenantApi call as well as direct queries against the package.
// The mix is intentional: with operator+DeleteCredential, the TenantApi
// package returns [] (no verb match), but data.smartcore.allow returns [true]
// via the catch-all operator+write rule.
var regoConsistencyCases = []regoConsistencyCase{
	{name: "tenant_operator_delete", query: "data.smartcore.bos.tenants.v1.TenantApi.allow", method: "DeleteCredential", role: "operator"},
	{name: "tenant_operator_add", query: "data.smartcore.bos.tenants.v1.TenantApi.allow", method: "AddTenant", role: "operator"},
	{name: "smartcore_operator_delete", query: "data.smartcore.allow", method: "DeleteCredential", role: "operator"},
	{name: "smartcore_operator_get", query: "data.smartcore.allow", method: "GetTenant", role: "operator"},
	{name: "smartcore_viewer_delete", query: "data.smartcore.allow", method: "DeleteCredential", role: "viewer"},
	{name: "smartcore_viewer_get", query: "data.smartcore.allow", method: "GetTenant", role: "viewer"},
	{name: "smartcore_admin_delete", query: "data.smartcore.allow", method: "DeleteCredential", role: "admin"},
	{name: "grpc_default_operator_delete", query: "data.grpc_default.allow", method: "DeleteCredential", role: "operator"},
}

func (c regoConsistencyCase) attrs() Attributes {
	return Attributes{
		Protocol:     ProtocolGRPC,
		Service:      "smartcore.bos.tenants.v1.TenantApi",
		Method:       c.method,
		Request:      json.RawMessage(`{"name":"test"}`),
		TokenPresent: true,
		TokenValid:   true,
		TokenClaims: token.Claims{
			SystemRoles: []string{c.role},
		},
	}
}

// resultSignature reduces a rego.ResultSet to a comparable string capturing
// only the bits Validate actually consumes: how many results there are and
// whether Allowed() is true. Using the full ResultSet is too strict because
// PartialResult-derived results carry different *ast.Location pointers than
// direct Eval, even when the boolean answer is identical.
func resultSignature(rs rego.ResultSet) string {
	return fmt.Sprintf("len=%d allowed=%t", len(rs), rs.Allowed())
}

func staticSignature(t *testing.T, query string, attrs Attributes) string {
	t.Helper()
	policy := &static{compiler: defaultCompiler}
	rs, err := policy.EvalPolicy(context.Background(), query, attrs)
	if err != nil {
		t.Fatalf("static EvalPolicy(%q) failed: %v", query, err)
	}
	return resultSignature(rs)
}

// TestCachedStaticMatchesStatic checks that the cached implementation produces
// the same result as the uncached one for every (query, input) pair, both when
// each cache entry is freshly populated and after the partial has been reused
// many times with different inputs. A divergence here indicates the cached
// PartialResult is leaking input-derived state across calls.
func TestCachedStaticMatchesStatic(t *testing.T) {
	expected := make(map[string]string, len(regoConsistencyCases))
	for _, c := range regoConsistencyCases {
		expected[c.name] = staticSignature(t, c.query, c.attrs())
	}

	check := func(t *testing.T, p Policy, c regoConsistencyCase) {
		t.Helper()
		rs, err := p.EvalPolicy(context.Background(), c.query, c.attrs())
		if err != nil {
			t.Fatalf("cached EvalPolicy(%q, %s) failed: %v", c.query, c.name, err)
		}
		if got := resultSignature(rs); got != expected[c.name] {
			t.Errorf("cached EvalPolicy(%q, %s)\n  got:  %s\n  want: %s", c.query, c.name, got, expected[c.name])
		}
	}

	t.Run("first_call", func(t *testing.T) {
		// Fresh cache for each case so we exercise the PartialResult on its
		// very first Eval against an input.
		for _, c := range regoConsistencyCases {
			c := c
			t.Run(c.name, func(t *testing.T) {
				p := newCachedStatic(defaultCompiler)
				check(t, p, c)
			})
		}
	})

	t.Run("repeated_mixed_inputs", func(t *testing.T) {
		// One shared cache. Each case is evaluated several times, interleaved,
		// so a given PartialResult sees many different inputs.
		p := newCachedStatic(defaultCompiler)
		const rounds = 20
		for i := 0; i < rounds; i++ {
			for _, c := range regoConsistencyCases {
				check(t, p, c)
			}
		}
	})

	t.Run("via_validate", func(t *testing.T) {
		// Validate walks the query hierarchy, which is how the original drift
		// was observed. We only run cases where the role/method combination
		// gives a stable Validate outcome we can reason about.
		p := newCachedStatic(defaultCompiler)
		ref := &static{compiler: defaultCompiler}
		for _, c := range regoConsistencyCases {
			c := c
			t.Run(c.name, func(t *testing.T) {
				attrs := c.attrs()
				_, refErr := Validate(context.Background(), ref, attrs)
				_, gotErr := Validate(context.Background(), p, attrs)
				if !errEqual(refErr, gotErr) {
					t.Errorf("Validate divergence for %s\n  cached:   %v\n  uncached: %v", c.name, gotErr, refErr)
				}
			})
		}
	})
}

// TestCachedStaticConcurrent hammers a single cachedStatic from many goroutines
// evaluating one query against many different inputs. Without serialisation
// inside cachedStatic.EvalPolicy, OPA's PartialResult mutates the shared
// *ast.Compiler concurrently and the test binary aborts with a Go runtime
// fatal ("concurrent map read and map write"). The lock in EvalPolicy makes
// this safe; if the lock is removed, this test will crash the binary.
//
// We deliberately use only one query here. Multiple queries on a shared cache
// hit a separate sequential drift bug (covered by the repeated_mixed_inputs
// subtest of TestCachedStaticMatchesStatic), which is independent of
// concurrency.
func TestCachedStaticConcurrent(t *testing.T) {
	const query = "data.smartcore.allow"

	cases := []regoConsistencyCase{
		{name: "operator_delete", query: query, method: "DeleteCredential", role: "operator"},
		{name: "operator_get", query: query, method: "GetTenant", role: "operator"},
		{name: "viewer_delete", query: query, method: "DeleteCredential", role: "viewer"},
		{name: "viewer_get", query: query, method: "GetTenant", role: "viewer"},
		{name: "admin_anything", query: query, method: "WhateverCredential", role: "admin"},
	}

	expected := make(map[string]string, len(cases))
	for _, c := range cases {
		expected[c.name] = staticSignature(t, c.query, c.attrs())
	}

	p := newCachedStatic(defaultCompiler)

	const (
		workers       = 32
		iterPerWorker = 100
	)

	var (
		wg       sync.WaitGroup
		mismatch sync.Once
		failMsg  string
	)

	wg.Add(workers)
	for w := 0; w < workers; w++ {
		w := w
		go func() {
			defer wg.Done()
			for i := 0; i < iterPerWorker; i++ {
				c := cases[(w+i)%len(cases)]
				rs, err := p.EvalPolicy(context.Background(), c.query, c.attrs())
				if err != nil {
					mismatch.Do(func() {
						failMsg = fmt.Sprintf("worker %d iter %d: EvalPolicy(%s) failed: %v", w, i, c.name, err)
					})
					return
				}
				if got := resultSignature(rs); got != expected[c.name] {
					mismatch.Do(func() {
						failMsg = fmt.Sprintf("worker %d iter %d: EvalPolicy(%s)\n  got:  %s\n  want: %s", w, i, c.name, got, expected[c.name])
					})
					return
				}
			}
		}()
	}
	wg.Wait()

	if failMsg != "" {
		t.Fatal(failMsg)
	}
}

// TestCachedStaticPartialResultReuse focuses on a single cached PartialResult:
// build it once via the first Eval, then drive it through alternating inputs
// many times, asserting each result still matches the uncached reference. If
// rego.PartialResult is not safe to reuse across calls with different inputs,
// this test will catch it without the noise of the rest of the cache.
func TestCachedStaticPartialResultReuse(t *testing.T) {
	const query = "data.smartcore.allow"

	cases := []regoConsistencyCase{
		{name: "operator_delete", query: query, method: "DeleteCredential", role: "operator"},
		{name: "operator_get", query: query, method: "GetTenant", role: "operator"},
		{name: "viewer_delete", query: query, method: "DeleteCredential", role: "viewer"},
		{name: "viewer_get", query: query, method: "GetTenant", role: "viewer"},
		{name: "admin_anything", query: query, method: "WhateverCredential", role: "admin"},
	}

	expected := make(map[string]string, len(cases))
	ref := &static{compiler: defaultCompiler}
	for _, c := range cases {
		rs, err := ref.EvalPolicy(context.Background(), c.query, c.attrs())
		if err != nil {
			t.Fatalf("static EvalPolicy(%s) failed: %v", c.name, err)
		}
		expected[c.name] = resultSignature(rs)
	}

	p := newCachedStatic(defaultCompiler)
	const rounds = 200
	for i := 0; i < rounds; i++ {
		for _, c := range cases {
			rs, err := p.EvalPolicy(context.Background(), c.query, c.attrs())
			if err != nil {
				t.Fatalf("round %d: cached EvalPolicy(%s) failed: %v", i, c.name, err)
			}
			if got := resultSignature(rs); got != expected[c.name] {
				t.Fatalf("round %d: cached EvalPolicy(%s) drifted\n  got:  %s\n  want: %s", i, c.name, got, expected[c.name])
			}
		}
	}
}

func errEqual(a, b error) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	case errors.Is(a, b) || errors.Is(b, a):
		return true
	default:
		// Compare gRPC status codes and messages for policy errors
		sa, oka := status.FromError(a)
		sb, okb := status.FromError(b)
		if oka && okb {
			return sa.Code() == sb.Code() && sa.Message() == sb.Message()
		}
		// Fall back to string comparison for non-status errors
		return a.Error() == b.Error()
	}
}
