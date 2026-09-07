// Package connect exposes the node's Smart Core Connect identity for
// authenticating to Connect services.
//
// This identity is distinct from the cohort identity carried by
// driver.Services.ClientTLSConfig and auto.Services.ClientTLSConfig - different
// keypair, different issuing CA, different lifecycle. Presenting the cohort
// identity to Connect will not authenticate.
package connect

import "crypto/tls"

// Credential is the node's Smart Core Connect identity under mTLS, together with
// the API origin it authenticates against. It is supplied by the node's cloud
// connection.
//
// Callers must handle three states:
//
//   - A nil Credential means no cloud connection is configured. Degrade, or fail
//     with a clear message, rather than dereference it.
//   - A non-nil Credential on a node that has not yet enrolled:
//     GetClientCertificate returns an error, and NodeID and APIEndpoint both
//     return "".
//   - An enrolled node: all three return live values.
//
// Every accessor reads the connection state per call, so a Credential follows
// certificate renewal and late enrolment without being re-fetched. That is what
// makes GetClientCertificate safe to install directly as
// tls.Config.GetClientCertificate.
type Credential interface {
	GetClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error)
	// NodeID returns the SCC node id (the leaf Subject CN), stable across
	// renewals. Empty when not enrolled.
	NodeID() string
	// APIEndpoint returns the SCC API origin (scheme://host) this credential
	// authenticates against. Empty when not enrolled.
	APIEndpoint() string
}
