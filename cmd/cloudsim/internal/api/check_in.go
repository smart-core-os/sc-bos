package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store"
	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store/queries"
)

// CheckInResponse is returned from the check-in endpoint.
type CheckInResponse struct {
	CheckIn      NodeCheckIn   `json:"checkIn"`
	LatestConfig *LatestConfig `json:"latestConfig,omitempty"`
}

type LatestConfig struct {
	Deployment    Deployment    `json:"deployment"`
	ConfigVersion ConfigVersion `json:"configVersion"`
}

// parseBearerSecret extracts a bearer token from the Authorization header,
// base64-decodes it, and returns the SHA-256 hash of the decoded secret.
func parseBearerSecret(r *http.Request) ([]byte, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, errors.New("missing authorization header")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return nil, errors.New("invalid authorization scheme")
	}
	token := auth[len(prefix):]
	if token == "" {
		return nil, errors.New("empty bearer token")
	}
	secret, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, errors.New("invalid base64 in bearer token")
	}
	hash := sha256.Sum256(secret)
	return hash[:], nil
}

func (s *Server) checkIn(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	secretHash, err := parseBearerSecret(r)
	if err != nil {
		writeError(w, errUnauthorized)
		logger.Debug("check-in auth failed", zap.Error(err))
		return
	}

	var (
		checkIn       queries.NodeCheckIn
		deployment    queries.Deployment
		configVersion queries.ConfigVersion
		hasDeploy     bool
	)
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		node, err := tx.GetNodeBySecretHash(r.Context(), secretHash)
		if err != nil {
			return err
		}

		checkIn, err = tx.CreateNodeCheckIn(r.Context(), node.ID)
		if err != nil {
			return err
		}

		deployment, err = tx.GetActiveDeploymentByNode(r.Context(), node.ID)
		if errors.Is(err, sql.ErrNoRows) {
			hasDeploy = false
			return nil
		}
		if err != nil {
			return err
		}
		hasDeploy = true
		configVersion, err = tx.GetConfigVersion(r.Context(), deployment.ConfigVersionID)
		return err
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Node not found by secret hash — return 401 to avoid revealing existence
			writeError(w, errUnauthorized)
			logger.Debug("check-in with unknown secret hash")
			return
		}
		writeError(w, errInternal)
		logger.Error("check-in failed", zap.Error(err))
		return
	}

	resp := CheckInResponse{
		CheckIn: toNodeCheckIn(checkIn),
	}
	if hasDeploy {
		resp.LatestConfig = &LatestConfig{
			Deployment:    toDeployment(deployment),
			ConfigVersion: toConfigVersion(r, configVersion),
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
