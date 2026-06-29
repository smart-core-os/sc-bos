package sim

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

// deviceRenew handles POST /v1/device/certificate/renew (EST simplereenroll in
// shape). It is authorised by the controller's current client certificate (mTLS),
// not an enrollment code. It signs the submitted CSR with the same CN (node id)
// and the same credentialId, and updates that slot's certificate metadata — the
// credentialId is unchanged so the broker (and slot) need not be touched.
func (s *Server) deviceRenew(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	dev, err := s.authenticateDevice(r)
	if err != nil {
		writeError(w, errUnauthorized)
		logger.Debug("renew auth failed", zap.Error(err))
		return
	}

	csr, err := readCSR(r)
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid CSR", zap.Error(err))
		return
	}

	pem, leaf, err := s.ca.signDevice(csr.PublicKey, formatID(dev.nodeID), dev.credentialID, defaultCertLifetime)
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to sign renewal", zap.Error(err))
		return
	}

	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, err := tx.UpdateCredentialCert(r.Context(), queries.UpdateCredentialCertParams{
			CredentialID: dev.credentialID,
			Serial:       serialHex(leaf),
			Fingerprint:  leafFingerprint(leaf),
			NotBefore:    leaf.NotBefore,
			NotAfter:     leaf.NotAfter,
		})
		return err
	})
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to record renewed certificate", zap.Error(err))
		return
	}

	// Renewal replaces an existing certificate: 200 OK (SCC returns 200 here).
	writeCertChain(w, http.StatusOK, pem)
}
