package authn

import (
	"errors"
	"fmt"
	"os"
	"path"

	"go.uber.org/multierr"

	"github.com/smart-core-os/sc-bos/internal/auth/accesstoken"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/system/authn/config"
)

func loadFileVerifier(idConfig *config.Identities, dataDirs []string, defaultFilename string) (accesstoken.Verifier, error) {
	// load identities from files in dataDirs
	ids, err := loadFileIdentities(idConfig, dataDirs, defaultFilename)
	if err != nil {
		return nil, fmt.Errorf("local accounts: %w", err)
	}

	// create a verifier from the loaded identities
	verifier, err := newStaticVerifier(ids)
	if err != nil {
		return nil, fmt.Errorf("local accounts: %w", err)
	}

	return verifier, nil
}

// loadFileIdentities loads identities from zero or more files in dataDirs.
// Creds are loaded from the JSON files that match `defaultFilename` in the `dataDirs` directory list and combined.
// No checks are made for duplicate IDs either within the same file or across multiple files.
func loadFileIdentities(idConfig *config.Identities, dataDirs []string, defaultFilename string) ([]config.Identity, error) {
	var ids []config.Identity
	for _, dataDir := range dataDirs {
		_, err := os.Stat(path.Join(dataDir, defaultFilename))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			} else {
				return nil, fmt.Errorf("local accounts: %w", err)
			}
		}
		i, err := idConfig.Load(dataDir, defaultFilename)
		if err != nil {
			return nil, fmt.Errorf("local accounts: %w", err)
		}
		ids = append(ids, i...)
	}
	return ids, nil
}

// newStaticVerifier returns a accesstoken.Verifier that checks credentials against the identities provided. No checks
// are made to prevent duplicate IDs. If ids is empty, a accesstoken.NeverVerify verifier is returned.
func newStaticVerifier(ids []config.Identity) (accesstoken.Verifier, error) {
	if len(ids) == 0 {
		return accesstoken.NeverVerify(errors.New("no local accounts")), nil
	}

	verifier := &accesstoken.MemoryVerifier{}
	var allErrs error
	for _, t := range ids {
		permissions := make([]token.PermissionAssignment, 0, len(t.Zones))
		for _, zone := range t.Zones {
			permissions = append(permissions, accesstoken.LegacyZonePermission(zone))
		}
		err := verifier.AddRecord(accesstoken.SecretData{Title: t.Title, TenantID: t.ID, Permissions: permissions, SystemRoles: t.Roles})
		if err != nil {
			allErrs = multierr.Append(allErrs, err)
			continue
		}
		for _, secret := range t.Secrets {
			_, err := verifier.AddSecretHash(t.ID, []byte(secret.Hash))
			allErrs = multierr.Append(allErrs, err)
		}
	}

	if allErrs != nil {
		return nil, fmt.Errorf("local accounts: %w", allErrs)
	}
	return verifier, nil
}
