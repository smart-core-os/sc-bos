package policy

import (
	"context"
	"encoding/json"
	"testing"

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
