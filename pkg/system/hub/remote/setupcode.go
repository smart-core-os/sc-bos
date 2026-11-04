package remote

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// setupCodeCreds presents a node's setup code as a bearer token, which a node with policy on that nobody has claimed
// accepts in place of a client certificate or a login.
type setupCodeCreds string

func (c setupCodeCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + string(c)}, nil
}

func (c setupCodeCreds) RequireTransportSecurity() bool {
	return true
}

// withSetupCode returns the call options that present code to the target, or none if code is empty.
// Attach them only to the calls that need the code: every wrong code a target sees uses up one of its limited guesses.
func withSetupCode(code string) []grpc.CallOption {
	if code == "" {
		return nil
	}
	return []grpc.CallOption{grpc.PerRPCCredentials(setupCodeCreds(code))}
}

// authError turns a target's Unauthenticated into PermissionDenied, saying what the setup code had to do with it.
// The hub's caller authenticated with the hub, not with the target, and the Ops UI logs the user out on
// Unauthenticated, so that code must not reach them.
// A target reports a missing, wrong or rate-limited code all the same way, so neither can the message tell them apart.
// Nor can it tell whether the target has a code at all: one that is claimed, or that has another way to log in, has none.
func authError(err error, setupCode string) error {
	if status.Code(err) != codes.Unauthenticated {
		return err
	}
	if setupCode == "" {
		return status.Error(codes.PermissionDenied, "the node refused the hub: if nobody has set the node up yet, give its setup code")
	}
	return status.Error(codes.PermissionDenied,
		"the node rejected the setup code: it is wrong, the node is already set up, or there were too many attempts")
}
