package sim

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

// parseBearerToken extracts the token from an "Authorization: Bearer <token>"
// header. Used by the registration endpoint, where the bearer token is the
// enrollment code.
func parseBearerToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", errors.New("missing authorization header")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return "", errors.New("invalid authorization scheme")
	}
	token := auth[len(prefix):]
	if token == "" {
		return "", errors.New("empty bearer token")
	}
	return token, nil
}

// authedDevice is the identity resolved from a verified client certificate.
type authedDevice struct {
	nodeID       int64
	credentialID string
}

// authenticateNode resolves the node id from the request's client certificate.
// See authenticateDevice for the full validation.
func (s *Server) authenticateNode(r *http.Request) (int64, error) {
	dev, err := s.authenticateDevice(r)
	if err != nil {
		return 0, err
	}
	return dev.nodeID, nil
}

// authenticateDevice validates the request's client certificate against live
// credential state. The TLS layer (ClientCAs + VerifyClientCertIfGiven) has
// already verified the chain to the Connect CA and the validity window; here we
// resolve CN -> node id and require the certificate's credentialId to still
// occupy one of that node's slots (this is where revocation takes effect: a
// retired credential's row is gone, so the lookup fails).
func (s *Server) authenticateDevice(r *http.Request) (authedDevice, error) {
	if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 {
		return authedDevice{}, errors.New("no verified client certificate")
	}
	leaf := r.TLS.PeerCertificates[0]

	nodeID, err := strconv.ParseInt(leaf.Subject.CommonName, 10, 64)
	if err != nil {
		return authedDevice{}, fmt.Errorf("certificate CN %q is not a node id: %w", leaf.Subject.CommonName, err)
	}
	credID := credentialIDFromCert(leaf)
	if credID == "" {
		return authedDevice{}, errors.New("certificate carries no credential id")
	}

	var cred queries.Credential
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var err error
		cred, err = tx.GetCredentialByCredentialID(r.Context(), credID)
		return err
	})
	if err != nil {
		return authedDevice{}, fmt.Errorf("credential no longer occupies a slot: %w", err)
	}
	if cred.NodeID != nodeID {
		return authedDevice{}, errors.New("credential does not belong to the certificate's node")
	}
	return authedDevice{nodeID: nodeID, credentialID: credID}, nil
}
