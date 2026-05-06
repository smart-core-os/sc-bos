package policy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/node/alltraits"
)

func mustNewPool(t *testing.T, modules map[string]*ast.Module) *pool {
	t.Helper()
	p, err := newPool(modules)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func mustNewCache(t *testing.T, modules map[string]*ast.Module) *cache {
	t.Helper()
	c, err := newCache(modules)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func mustNewStatic(t *testing.T, modules map[string]*ast.Module) *static {
	t.Helper()
	s, err := newStatic(modules)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

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

	b.Run("pool", func(b *testing.B) {
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

	t.Run("pool", func(t *testing.T) {
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
	policy := mustNewStatic(t, defaultModules)
	rs, err := policy.EvalPolicy(context.Background(), query, attrs)
	if err != nil {
		t.Fatalf("static EvalPolicy(%q) failed: %v", query, err)
	}
	return resultSignature(rs)
}

// TestPoolMatchesStatic checks that the pool implementation produces the same
// result as the uncached static one for every (query, input) pair, both when
// each cache entry is freshly populated and after the partial has been reused
// many times with different inputs. A divergence here indicates a PartialResult
// is leaking input-derived state across calls.
func TestPoolMatchesStatic(t *testing.T) {
	expected := make(map[string]string, len(regoConsistencyCases))
	for _, c := range regoConsistencyCases {
		expected[c.name] = staticSignature(t, c.query, c.attrs())
	}

	check := func(t *testing.T, p Policy, c regoConsistencyCase) {
		t.Helper()
		rs, err := p.EvalPolicy(context.Background(), c.query, c.attrs())
		if err != nil {
			t.Fatalf("pool EvalPolicy(%q, %s) failed: %v", c.query, c.name, err)
		}
		if got := resultSignature(rs); got != expected[c.name] {
			t.Errorf("pool EvalPolicy(%q, %s)\n  got:  %s\n  want: %s", c.query, c.name, got, expected[c.name])
		}
	}

	t.Run("first_call", func(t *testing.T) {
		// Fresh pool for each case so we exercise the PartialResult on its
		// very first Eval against an input.
		for _, c := range regoConsistencyCases {
			t.Run(c.name, func(t *testing.T) {
				p := mustNewPool(t, defaultModules)
				check(t, p, c)
			})
		}
	})

	t.Run("repeated_mixed_inputs", func(t *testing.T) {
		// One shared pool. Each case is evaluated several times, interleaved,
		// so a given PartialResult sees many different inputs.
		p := mustNewPool(t, defaultModules)
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
		p := mustNewPool(t, defaultModules)
		ref := mustNewStatic(t, defaultModules)
		for _, c := range regoConsistencyCases {
			t.Run(c.name, func(t *testing.T) {
				attrs := c.attrs()
				_, refErr := Validate(context.Background(), ref, attrs)
				_, gotErr := Validate(context.Background(), p, attrs)
				if !errEqual(refErr, gotErr) {
					t.Errorf("Validate divergence for %s\n  pool:     %v\n  static:   %v", c.name, gotErr, refErr)
				}
			})
		}
	})
}

// TestPoolConcurrent hammers a single pool from many goroutines evaluating one
// query against many different inputs. The pool assigns each goroutine its own
// *cache from a sync.Pool, so there is no shared mutable PartialResult state
// between concurrent callers. If pool.Get/Put or *cache reuse is broken, this
// test will surface it via result mismatches or a runtime crash.
func TestPoolConcurrent(t *testing.T) {
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

	p := mustNewPool(t, defaultModules)

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

// TestCachePartialResultReuse focuses on a single *cache: build its PartialResult
// for a query once via the first EvalPolicy call, then drive it through
// alternating inputs many times, asserting each result still matches the static
// reference. If PartialResult is not safe to reuse across sequential calls with
// different inputs, this test will catch it.
func TestCachePartialResultReuse(t *testing.T) {
	const query = "data.smartcore.allow"

	cases := []regoConsistencyCase{
		{name: "operator_delete", query: query, method: "DeleteCredential", role: "operator"},
		{name: "operator_get", query: query, method: "GetTenant", role: "operator"},
		{name: "viewer_delete", query: query, method: "DeleteCredential", role: "viewer"},
		{name: "viewer_get", query: query, method: "GetTenant", role: "viewer"},
		{name: "admin_anything", query: query, method: "WhateverCredential", role: "admin"},
	}

	expected := make(map[string]string, len(cases))
	ref := mustNewStatic(t, defaultModules)
	for _, c := range cases {
		rs, err := ref.EvalPolicy(context.Background(), c.query, c.attrs())
		if err != nil {
			t.Fatalf("static EvalPolicy(%s) failed: %v", c.name, err)
		}
		expected[c.name] = resultSignature(rs)
	}

	c := mustNewCache(t, defaultModules)
	const rounds = 200
	for i := 0; i < rounds; i++ {
		for _, tc := range cases {
			rs, err := c.EvalPolicy(context.Background(), tc.query, tc.attrs())
			if err != nil {
				t.Fatalf("round %d: cache EvalPolicy(%s) failed: %v", i, tc.name, err)
			}
			if got := resultSignature(rs); got != expected[tc.name] {
				t.Fatalf("round %d: cache EvalPolicy(%s) drifted\n  got:  %s\n  want: %s", i, tc.name, got, expected[tc.name])
			}
		}
	}
}

// TestCacheContextCancelDoesNotPoisonCache verifies that when getOrCreateCacheEntry
// is called with a cancelled context, the error entry is not stored in the cache.
// A subsequent call with a live context must succeed and produce a correct result.
func TestCacheContextCancelDoesNotPoisonCache(t *testing.T) {
	const query = "data.smartcore.allow"
	attrs := regoConsistencyCase{name: "admin", query: query, method: "GetTenant", role: "admin"}.attrs()

	c := mustNewCache(t, defaultModules)

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// First call with a cancelled context: EvalPolicy should fail.
	_, err := c.EvalPolicy(cancelledCtx, query, attrs)
	if err == nil {
		t.Fatal("expected error from EvalPolicy with cancelled context, got nil")
	}

	// Second call with a live context: should succeed because the failure was not cached.
	rs, err := c.EvalPolicy(context.Background(), query, attrs)
	if err != nil {
		t.Fatalf("EvalPolicy after cancelled context failed: %v (cache was poisoned)", err)
	}
	want := staticSignature(t, query, attrs)
	if got := resultSignature(rs); got != want {
		t.Errorf("EvalPolicy after cancelled context: got %s, want %s", got, want)
	}
}

// TestCacheEvalPolicyPropagatesError verifies that when PartialResult fails for a
// non-context reason (e.g. an invalid query), the error is cached and EvalPolicy
// returns it directly rather than trying to evaluate a zero-value PartialResult.
func TestCacheEvalPolicyPropagatesError(t *testing.T) {
	// An invalid query expression that PartialResult will reject.
	const badQuery = "1 +"

	c := mustNewCache(t, defaultModules)
	attrs := Attributes{}

	// First call: PartialResult fails and caches the error.
	_, firstErr := c.EvalPolicy(context.Background(), badQuery, attrs)
	if firstErr == nil {
		t.Fatal("expected error from EvalPolicy with bad query, got nil")
	}

	// Second call with the same query: must return the same error, not a
	// confusing nil-compiler error from a zero-value PartialResult.
	_, secondErr := c.EvalPolicy(context.Background(), badQuery, attrs)
	if secondErr == nil {
		t.Fatal("expected error on second EvalPolicy call with bad query, got nil")
	}
	if firstErr.Error() != secondErr.Error() {
		t.Errorf("error changed between calls:\n  first:  %v\n  second: %v", firstErr, secondErr)
	}
}

// TestNewCacheError verifies that newCache, newPool, and newStatic all return
// errors when given modules that fail compilation.
func TestNewCacheError(t *testing.T) {
	// A complete rule and a partial-set rule with the same name conflict under
	// OPA v1 strict typing (rego_type_error: conflicting rules).
	mod, err := ast.ParseModule("bad.rego", `
package bad
import rego.v1
p := 1
p contains "x" if true
`)
	if err != nil {
		t.Fatalf("ParseModule: %v", err)
	}
	badModules := map[string]*ast.Module{"bad.rego": mod}

	t.Run("newCache", func(t *testing.T) {
		_, err := newCache(badModules)
		if err == nil {
			t.Fatal("expected error from newCache with conflicting rules, got nil")
		}
	})

	t.Run("newPool", func(t *testing.T) {
		_, err := newPool(badModules)
		if err == nil {
			t.Fatal("expected error from newPool with conflicting rules, got nil")
		}
	})

	t.Run("newStatic", func(t *testing.T) {
		_, err := newStatic(badModules)
		if err == nil {
			t.Fatal("expected error from newStatic with conflicting rules, got nil")
		}
	})
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
