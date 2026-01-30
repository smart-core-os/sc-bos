package accesstoken

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/proto/tenantpb"
)

// RemoteVerifier implements Verifier by calling TenantApiClient.VerifySecret.
type RemoteVerifier struct {
	Client tenantpb.TenantApiClient
}

func (r *RemoteVerifier) Verify(ctx context.Context, id, secret string) (SecretData, error) {
	return RemoteVerify(ctx, id, secret, r.Client)
}

// RemoteVerify verifies that id and secret are a valid pair using client.
func RemoteVerify(ctx context.Context, id, secret string, client tenantpb.TenantApiClient) (SecretData, error) {
	fail := func(e error) (SecretData, error) {
		return SecretData{}, e
	}

	// we cancel the context in the case where the VerifySecret call completes with failure before the GetTenant call completes
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Get the tenant info in parallel, we don't use it if the verify request fails, but we think it's better to reduce
	// latency for the verify call over doing extra requests on the server in the case of failure.
	type getResponse struct {
		t *tenantpb.Tenant
		e error
	}
	getC := make(chan getResponse, 1)
	go func() {
		var r getResponse
		r.t, r.e = client.GetTenant(ctx, &tenantpb.GetTenantRequest{Id: id})
		getC <- r
	}()

	_, err := client.VerifySecret(ctx, &tenantpb.VerifySecretRequest{TenantId: id, Secret: secret})
	if err != nil {
		return fail(err)
	}

	// wait for either the GetTenant request or the context to complete
	select {
	case r := <-getC:
		if r.e != nil {
			return fail(r.e)
		}
		permissions := make([]token.PermissionAssignment, 0, len(r.t.ZoneNames))
		for _, zone := range r.t.ZoneNames {
			permissions = append(permissions, LegacyZonePermission(zone))
		}
		return SecretData{
			TenantID:    id,
			Permissions: permissions,
		}, nil
	case <-ctx.Done():
		return fail(ctx.Err())
	}
}
