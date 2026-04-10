package authn

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/account"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/system/authn/config"
)

// TestApplyConfig_ImportError verifies that when importIdentities fails (here: cancelled context),
// applyConfig propagates the error instead of silently swallowing it.
func TestApplyConfig_ImportError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := account.NewMemoryStore(logger)
	validators := new(token.ValidatorSet)

	s := &System{
		logger:     logger,
		accounts:   store,
		validators: validators,
		server:     &nextOrNotFound{},
	}

	// Construct a FileAccounts with inline content to bypass file loading.
	var fileAccounts config.Identities
	if err := json.Unmarshal([]byte(`[]`), &fileAccounts); err != nil {
		t.Fatalf("failed to construct file accounts: %v", err)
	}

	cfg := config.Root{
		User: &config.User{
			// LocalAccounts=true sets localAccountsAvailable=true, which is a precondition for ImportFileAccounts.
			LocalAccounts:      true,
			ImportFileAccounts: true,
			FileAccounts:       &fileAccounts,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling so importIdentities fails

	err := s.applyConfig(ctx, cfg)
	if err == nil {
		t.Fatal("expected error when context is cancelled during import, got nil")
	}
}
