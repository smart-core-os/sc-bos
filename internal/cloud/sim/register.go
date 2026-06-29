package sim

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

const enrollmentCodeExpiry = 15 * time.Minute

// Media types for the device enrolment exchange (inspired by EST, not an EST implementation).
const (
	mediaTypePKCS10    = "application/pkcs10"                // request: base64 DER CSR
	mediaTypeCertChain = "application/pem-certificate-chain" // response: PEM chain, leaf first
)

const maxCSRBodySize = 16 * 1024 // a base64 P-256 CSR is well under this

// EnrollmentCode is the JSON representation of an enrollment code.
type EnrollmentCode struct {
	ID         int64     `json:"id,string"`
	NodeID     int64     `json:"nodeId,string"`
	Code       string    `json:"code"`
	TargetSlot string    `json:"targetSlot"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

func toEnrollmentCode(ec queries.EnrollmentCode) EnrollmentCode {
	return EnrollmentCode{
		ID:         ec.ID,
		NodeID:     ec.NodeID,
		Code:       ec.Code,
		TargetSlot: ec.TargetSlot,
		ExpiresAt:  ec.ExpiresAt,
	}
}

// GenerateEnrollmentCode generates a random 6-character uppercase alphanumeric string.
func GenerateEnrollmentCode() string {
	return rand.Text()[:6]
}

// createEnrollmentCode handles POST /api/v1/management/nodes/{id}/enrollment-codes.
// It generates a new single-use enrollment code for the given node with a
// 15-minute expiry. An optional ?targetSlot=primary|secondary (default primary)
// selects which credential slot the exchange fills.
func (s *Server) createEnrollmentCode(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	nodeID, err := parseID(r.PathValue("id"))
	if err != nil || nodeID == 0 {
		writeError(w, errInvalidRequest)
		logger.Info("invalid node id", zap.Error(err))
		return
	}

	targetSlot := r.URL.Query().Get("targetSlot")
	if targetSlot == "" {
		targetSlot = slotPrimary
	}
	if targetSlot != slotPrimary && targetSlot != slotSecondary {
		writeError(w, errInvalidRequest)
		logger.Info("invalid target slot", zap.String("targetSlot", targetSlot))
		return
	}

	code := GenerateEnrollmentCode()
	expiresAt := time.Now().Add(enrollmentCodeExpiry)

	var item queries.EnrollmentCode
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		if _, err := tx.GetNode(r.Context(), nodeID); err != nil {
			return err
		}
		item, err = tx.CreateEnrollmentCode(r.Context(), queries.CreateEnrollmentCodeParams{
			NodeID:     nodeID,
			Code:       code,
			ExpiresAt:  expiresAt,
			TargetSlot: targetSlot,
		})
		return err
	})
	if err != nil {
		resErr := translateDBError(err)
		writeError(w, resErr)
		if resErr.internal() {
			logger.Error("failed to create enrollment code", zap.Error(err))
		} else {
			logger.Debug("bad request to create enrollment code", zap.Error(err))
		}
		return
	}

	writeJSON(w, http.StatusCreated, toEnrollmentCode(item))
}

// deviceRegister handles POST /v1/device/register (inspired by EST simpleenroll, not an EST implementation).
// The enrollment code is the bearer authorisation and the body is a base64 DER
// PKCS#10 CSR. It mints a credentialId, signs the leaf (CN = node id), fills the
// code's target slot, and returns the PEM certificate chain.
func (s *Server) deviceRegister(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	code, err := parseBearerToken(r)
	if err != nil {
		writeError(w, errUnauthorized)
		logger.Info("missing or invalid authorization header", zap.Error(err))
		return
	}

	csr, err := readCSR(r)
	if err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid CSR", zap.Error(err))
		return
	}

	credentialID, err := mintCredentialID()
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to mint credential id", zap.Error(err))
		return
	}

	var chainPEM []byte
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		ec, err := tx.GetActiveEnrollmentCode(r.Context(), code)
		if err != nil {
			return err
		}
		node, err := tx.GetNode(r.Context(), ec.NodeID)
		if err != nil {
			return err
		}

		pem, leaf, err := s.ca.signDevice(csr.PublicKey, formatID(node.ID), credentialID, defaultCertLifetime)
		if err != nil {
			return err
		}
		chainPEM = pem

		if _, err := tx.UpsertCredential(r.Context(), queries.UpsertCredentialParams{
			NodeID:       node.ID,
			CredentialID: credentialID,
			Slot:         ec.TargetSlot,
			Serial:       serialHex(leaf),
			Fingerprint:  leafFingerprint(leaf),
			NotBefore:    leaf.NotBefore,
			NotAfter:     leaf.NotAfter,
		}); err != nil {
			return err
		}
		return tx.MarkEnrollmentCodeUsed(r.Context(), ec.ID)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Mirror SCC: a bad/expired/used code is a generic 401 "unauthorized",
			// not a distinct code — auth failures never reveal whether the code existed.
			writeError(w, errUnauthorized)
			logger.Info("enrollment code invalid, expired, or already used", zap.String("code", code))
			return
		}
		writeError(w, errInternal)
		logger.Error("failed to register device", zap.Error(err))
		return
	}

	// Registration issues the first certificate: 201 Created (SCC returns 201 here).
	writeCertChain(w, http.StatusCreated, chainPEM)
}

// readCSR reads a base64 DER PKCS#10 CSR from the request body and parses it
// (verifying its signature). The body is base64-encoded DER, matching the
// application/pkcs10 content type.
func readCSR(r *http.Request) (*x509.CertificateRequest, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxCSRBodySize))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	der, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(body)))
	if err != nil {
		return nil, fmt.Errorf("base64 decode CSR: %w", err)
	}
	return pki.ParseCSRDER(der)
}

// writeCertChain writes a PEM certificate chain response with the given status
// (201 for registration's first issuance, 200 for renewal — mirroring SCC).
func writeCertChain(w http.ResponseWriter, status int, chainPEM []byte) {
	w.Header().Set("Content-Type", mediaTypeCertChain)
	w.WriteHeader(status)
	_, _ = w.Write(chainPEM)
}

// formatID formats an int64 ID as a decimal string, matching the json:"id,string" convention.
func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}
