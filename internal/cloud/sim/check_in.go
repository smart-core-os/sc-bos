package sim

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	queries2 "github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

// CheckInResponse is returned from the check-in endpoint.
type CheckInResponse struct {
	CheckIn      CheckInAck    `json:"checkIn"`
	LatestConfig *LatestConfig `json:"latestConfig,omitempty"`
}

// CheckInAck is the acknowledgement portion of the check-in response.
type CheckInAck struct {
	NodeID      int64     `json:"nodeId,string"`
	CheckInTime time.Time `json:"checkInTime"`
}

type LatestConfig struct {
	Deployment    Deployment    `json:"deployment"`
	ConfigVersion ConfigVersion `json:"configVersion"`
}

// CheckInRequest is the optional request body for the check-in endpoint.
type CheckInRequest struct {
	CurrentDeployment    *CheckInDeploymentRef        `json:"currentDeployment,omitempty"`
	InstallingDeployment *CheckInInstallingDeployment `json:"installingDeployment,omitempty"`
	FailedDeployment     *CheckInFailedDeployment     `json:"failedDeployment,omitempty"`
}

// CheckInInstallingDeployment references a deployment being installed, optionally with error and attempt info.
type CheckInInstallingDeployment struct {
	ID       int64  `json:"id,string"`
	Error    string `json:"error,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
}

// CheckInDeploymentRef references a deployment by ID.
type CheckInDeploymentRef struct {
	ID int64 `json:"id,string"`
}

// CheckInFailedDeployment reports a deployment that failed, triggering a FAILED status update.
type CheckInFailedDeployment struct {
	ID     int64  `json:"id,string"`
	Reason string `json:"reason,omitempty"`
}

func (s *Server) checkIn(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	nodeID, err := s.authenticateNode(r)
	if err != nil {
		writeError(w, errUnauthorized)
		logger.Debug("check-in auth failed", zap.Error(err))
		return
	}

	var req CheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, errInvalidRequest)
		logger.Info("invalid json", zap.Error(err))
		return
	}

	var (
		checkIn       queries2.NodeCheckIn
		deployment    queries2.Deployment
		configVersion queries2.ConfigVersion
		hasDeploy     bool
	)
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		node, err := tx.GetNode(r.Context(), nodeID)
		if err != nil {
			return err
		}

		if req.InstallingDeployment != nil {
			// auto-update deployment status to IN_PROGRESS when node reports it's installing, if it was PENDING before
			row, err := tx.GetDeploymentWithConfigVersion(r.Context(), req.InstallingDeployment.ID)
			if errors.Is(err, sql.ErrNoRows) {
				return errInvalidRequest
			}
			if err != nil {
				return err
			}
			if row.NodeID != node.ID {
				return errInvalidRequest
			}
			if row.Status == statusPending {
				_, err = tx.UpdateDeploymentStatus(r.Context(), queries2.UpdateDeploymentStatusParams{
					ID:     req.InstallingDeployment.ID,
					Status: statusInProgress,
				})
				if err != nil {
					return err
				}
			}
		}

		if req.CurrentDeployment != nil {
			// auto-update deployment status to COMPLETED if node reports its current version
			row, err := tx.GetDeploymentWithConfigVersion(r.Context(), req.CurrentDeployment.ID)
			if errors.Is(err, sql.ErrNoRows) {
				return errInvalidRequest
			}
			if err != nil {
				return err
			}
			if row.NodeID != node.ID {
				return errInvalidRequest
			}
			if row.Status == statusInProgress {
				_, err = tx.UpdateDeploymentStatus(r.Context(), queries2.UpdateDeploymentStatusParams{
					ID:     req.CurrentDeployment.ID,
					Status: statusCompleted,
				})
				if err != nil {
					return err
				}
			}
		}

		if req.FailedDeployment != nil {
			row, err := tx.GetDeploymentWithConfigVersion(r.Context(), req.FailedDeployment.ID)
			if errors.Is(err, sql.ErrNoRows) {
				return errInvalidRequest
			}
			if err != nil {
				return err
			}
			if row.NodeID != node.ID {
				return errInvalidRequest
			}
			_, err = tx.UpdateDeploymentStatus(r.Context(), queries2.UpdateDeploymentStatusParams{
				ID:     req.FailedDeployment.ID,
				Status: statusFailed,
				Reason: nullString(req.FailedDeployment.Reason),
			})
			if err != nil {
				return err
			}
		}

		var currentID *int64
		if req.CurrentDeployment != nil {
			currentID = &req.CurrentDeployment.ID
		}
		var installingID *int64
		if req.InstallingDeployment != nil {
			installingID = &req.InstallingDeployment.ID
		}
		var installingError string
		var installingAttempts int
		if req.InstallingDeployment != nil {
			installingError = req.InstallingDeployment.Error
			installingAttempts = req.InstallingDeployment.Attempts
		}
		checkIn, err = tx.CreateNodeCheckIn(r.Context(), queries2.CreateNodeCheckInParams{
			NodeID:                       node.ID,
			CurrentDeploymentID:          nullInt64(currentID),
			InstallingDeploymentID:       nullInt64(installingID),
			InstallingDeploymentError:    nullString(installingError),
			InstallingDeploymentAttempts: nullInt64FromInt(installingAttempts),
		})
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
			writeError(w, errNotFound)
			logger.Debug("check-in for unknown node")
			return
		}
		if errors.Is(err, errInvalidRequest) {
			writeError(w, errInvalidRequest)
			logger.Info("invalid check-in request")
			return
		}
		writeError(w, errInternal)
		logger.Error("check-in failed", zap.Error(err))
		return
	}

	resp := CheckInResponse{
		CheckIn: CheckInAck{
			NodeID:      checkIn.NodeID,
			CheckInTime: checkIn.CheckInTime,
		},
	}
	if hasDeploy {
		resp.LatestConfig = &LatestConfig{
			Deployment:    toDeployment(deployment),
			ConfigVersion: toConfigVersion(r, configVersion),
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
