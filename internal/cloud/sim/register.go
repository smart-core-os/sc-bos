package sim

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

const enrollmentCodeExpiry = 15 * time.Minute

// EnrollmentCode is the JSON representation of an enrollment code.
type EnrollmentCode struct {
	ID        int64     `json:"id,string"`
	NodeID    int64     `json:"nodeId,string"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func toEnrollmentCode(ec queries.EnrollmentCode) EnrollmentCode {
	return EnrollmentCode{
		ID:        ec.ID,
		NodeID:    ec.NodeID,
		Code:      ec.Code,
		ExpiresAt: ec.ExpiresAt,
	}
}

// DeviceRegisterRequest is the body for POST /v1/device/register.
type DeviceRegisterRequest struct {
	ClientName string `json:"client_name"`
}

// DeviceRegisterResponse is the response for POST /v1/device/register.
type DeviceRegisterResponse struct {
	ClientID     string `json:"client_id"`
	ClientSecret []byte `json:"client_secret"` // automatically base64-encoded
	ClientName   string `json:"client_name"`
	BosapiRoot   string `json:"bosapi_root"`
}

// generateEnrollmentCode generates a random 6-character uppercase alphanumeric string.
func generateEnrollmentCode() string {
	return rand.Text()[:6]
}

// createEnrollmentCode handles POST /api/v1/management/nodes/{id}/enrollment-codes.
// It generates a new single-use enrollment code for the given node with a 15-minute expiry.
func (s *Server) createEnrollmentCode(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	nodeID, err := parseID(r.PathValue("id"))
	if err != nil || nodeID == 0 {
		writeError(w, errInvalidRequest)
		logger.Info("invalid node id", zap.Error(err))
		return
	}

	code := generateEnrollmentCode()
	expiresAt := time.Now().Add(enrollmentCodeExpiry)

	var item queries.EnrollmentCode
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		// Validate the node exists before creating the code.
		_, err := tx.GetNode(r.Context(), nodeID)
		if err != nil {
			return err
		}
		item, err = tx.CreateEnrollmentCode(r.Context(), queries.CreateEnrollmentCodeParams{
			NodeID:    nodeID,
			Code:      code,
			ExpiresAt: expiresAt,
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

// deviceRegister handles POST /v1/device/register.
// It exchanges a valid enrollment code for client credentials, updating the node's secret.
func (s *Server) deviceRegister(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	code, err := parseBearerToken(r)
	if err != nil {
		writeError(w, errUnauthorized)
		logger.Info("missing or invalid authorization header", zap.Error(err))
		return
	}

	var req DeviceRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errInvalidRequest)
		logger.Info("invalid json", zap.Error(err))
		return
	}

	secret, hash, err := generateSecret()
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to generate secret", zap.Error(err))
		return
	}

	var node queries.Node
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		ec, err := tx.GetActiveEnrollmentCode(r.Context(), code)
		if err != nil {
			return err
		}

		node, err = tx.GetNode(r.Context(), ec.NodeID)
		if err != nil {
			return err
		}

		err = tx.UpdateNodeSecretHash(r.Context(), queries.UpdateNodeSecretHashParams{
			ID:         node.ID,
			SecretHash: hash,
		})
		if err != nil {
			return err
		}

		return tx.MarkEnrollmentCodeUsed(r.Context(), ec.ID)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, errInvalidEnrollmentCode)
			logger.Info("enrollment code invalid, expired, or already used", zap.String("code", code))
			return
		}
		writeError(w, errInternal)
		logger.Error("failed to register device", zap.Error(err))
		return
	}

	writeJSON(w, http.StatusCreated, DeviceRegisterResponse{
		ClientID:     formatID(node.ID),
		ClientSecret: secret,
		ClientName:   req.ClientName,
		BosapiRoot:   sameHostURL(r, "/"),
	})
}

// formatID formats an int64 ID as a decimal string, matching the json:"id,string" convention.
func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}
