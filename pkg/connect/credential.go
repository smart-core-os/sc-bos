// Package connect exposes the node's Smart Core Connect identity for
// authenticating to Connect services.
//
// This identity is distinct from the cohort identity carried by
// driver.Services.ClientTLSConfig and auto.Services.ClientTLSConfig - different
// keypair, different issuing CA, different lifecycle.
package connect

import (
	"context"
	"crypto/tls"
	"errors"
)

var errStateClosed = errors.New("connect: state channel closed before the credential connected")

// Connectivity is the lifecycle state of the node's Connect enrolment.
type Connectivity int

const (
	NotEnrolled Connectivity = iota // no credential; GetClientCertificate errors
	Connecting                      // enrolled, no successful check-in yet
	Connected                       // last check-in succeeded
	Failed                          // last check-in failed
)

// State is a snapshot of the node's Connect identity and connection health.
// NodeID and APIEndpoint are empty iff Connectivity is NotEnrolled.
type State struct {
	Connectivity Connectivity
	// NodeID is the SCC node id (the leaf Subject CN), stable across renewals.
	NodeID string
	// APIEndpoint is the SCC API origin (scheme://host) the credential
	// authenticates against.
	APIEndpoint string
}

// Credential is the node's Smart Core Connect identity under mTLS, together with
// the API origin it authenticates against. It is supplied by the node's cloud
// connection.
//
// A nil Credential means the host has no cloud connection at all; degrade, or
// fail with a clear message, rather than dereference it. Otherwise the
// Credential is non-nil whether or not the node is enrolled.
//
// Every accessor reads the connection state per call, so a Credential follows
// certificate renewal and enrolment without being re-fetched. That is enough
// for callers that look up the identity on each use, and it is what makes
// GetClientCertificate safe to install directly as
// tls.Config.GetClientCertificate.
//
// Callers that hold a long-lived connection (MQTT, a gRPC stream) must also
// watch PullState: the node can be unlinked, or re-enrolled under a different
// node id or endpoint, at any time. Tear the connection down and reconnect when
// NodeID or APIEndpoint changes, and stop using it when Connectivity drops to
// NotEnrolled. A certificate renewal leaves State unchanged and needs no
// reconnect.
type Credential interface {
	// GetClientCertificate returns the current Connect leaf certificate, or an
	// error when the node is not enrolled.
	GetClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error)
	// State returns the current state snapshot.
	State() State
	// PullState returns the current state plus a channel that receives
	// subsequent changes, closed when ctx is done. A reader that falls behind
	// receives the latest state rather than every one in between, and never a
	// state equal to the last one it received.
	PullState(ctx context.Context) (State, <-chan State)
}

// WaitConnected blocks until cred reports Connected, or ctx is done. It returns
// nil once connected, otherwise ctx.Err() - or an error if cred closes its state
// channel early. If cred is already Connected it returns immediately.
func WaitConnected(ctx context.Context, cred Credential) error {
	// Scope the subscription to a child ctx we cancel on return, so it ends
	// with this call.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	state, changes := cred.PullState(ctx)
	if state.Connectivity == Connected {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case st, ok := <-changes:
			if !ok {
				if err := ctx.Err(); err != nil {
					return err
				}
				return errStateClosed
			}
			if st.Connectivity == Connected {
				return nil
			}
		}
	}
}
