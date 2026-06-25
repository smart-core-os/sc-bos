package sim

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

// CheckInResponse is returned from the check-in endpoint.
type CheckInResponse struct {
	CheckIn      CheckInAck    `json:"checkIn"`
	LatestConfig *LatestConfig `json:"latestConfig,omitempty"`
	LatestUpdate *LatestUpdate `json:"latestUpdate,omitempty"`
	// LatestSupervisorUpdate is the active supervisor-rpm update, a channel parallel to LatestUpdate
	// (BOS images). Added additively: existing fields are unchanged.
	LatestSupervisorUpdate *LatestUpdate `json:"latestSupervisorUpdate,omitempty"`
}

// LatestUpdate is the active update deployment and its artefact, returned on check-in. It is the
// software-update analogue of LatestConfig, used for both the BOS-image and supervisor-rpm channels.
type LatestUpdate struct {
	UpdateDeployment UpdateDeployment `json:"updateDeployment"`
	UpdateArtefact   UpdateArtefact   `json:"updateArtefact"`
}

// CheckInAck is the acknowledgement portion of the check-in response.
type CheckInAck struct {
	NodeID      int64     `json:"nodeId,string"`
	CheckInTime time.Time `json:"checkInTime"`
}

type LatestConfig struct {
	ConfigDeployment ConfigDeployment `json:"deployment"`
	ConfigVersion    ConfigVersion    `json:"configVersion"`
}

// CheckInRequest is the optional request body for the check-in endpoint.
//
// The config channel (currentDeployment/installingDeployment/failedDeployment) is the original,
// frozen wire contract. The update channel (currentUpdate/installingUpdate/failedUpdate) is the
// parallel software-update reporting, added without changing the config fields.
type CheckInRequest struct {
	CurrentDeployment    *CheckInDeploymentRef        `json:"currentDeployment,omitempty"`
	InstallingDeployment *CheckInInstallingDeployment `json:"installingDeployment,omitempty"`
	FailedDeployment     *CheckInFailedDeployment     `json:"failedDeployment,omitempty"`

	CurrentUpdate    *CheckInDeploymentRef    `json:"currentUpdate,omitempty"`
	InstallingUpdate *CheckInInstallingUpdate `json:"installingUpdate,omitempty"`
	FailedUpdate     *CheckInFailedUpdate     `json:"failedUpdate,omitempty"`
}

// CheckInInstallingUpdate references an update deployment being installed, optionally with error and
// attempt info.
type CheckInInstallingUpdate struct {
	ID       int64  `json:"id,string"`
	Error    string `json:"error,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
}

// CheckInFailedUpdate reports an update deployment that failed, triggering a FAILED status update.
type CheckInFailedUpdate struct {
	ID     int64  `json:"id,string"`
	Reason string `json:"reason,omitempty"`
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
		checkIn       queries.NodeCheckIn
		deployment    queries.ConfigDeployment
		configVersion queries.ConfigVersion
		hasDeploy     bool

		updateDep      queries.UpdateDeployment
		updateArtefact queries.UpdateArtefact
		hasUpdate      bool

		supUpdateDep      queries.UpdateDeployment
		supUpdateArtefact queries.UpdateArtefact
		hasSupUpdate      bool
	)
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		node, err := tx.GetNode(r.Context(), nodeID)
		if err != nil {
			return err
		}

		if req.InstallingDeployment != nil {
			// auto-update deployment status to IN_PROGRESS when node reports it's installing, if it was PENDING before
			row, err := tx.GetConfigDeploymentWithConfigVersion(r.Context(), req.InstallingDeployment.ID)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("installing deployment %d: not found: %w", req.InstallingDeployment.ID, errInvalidRequest)
			}
			if err != nil {
				return err
			}
			if row.NodeID != node.ID {
				return fmt.Errorf("installing deployment %d: belongs to a different node: %w", req.InstallingDeployment.ID, errInvalidRequest)
			}
			if row.Status == statusPending {
				_, err = tx.UpdateConfigDeploymentStatus(r.Context(), queries.UpdateConfigDeploymentStatusParams{
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
			row, err := tx.GetConfigDeploymentWithConfigVersion(r.Context(), req.CurrentDeployment.ID)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("current deployment %d: not found: %w", req.CurrentDeployment.ID, errInvalidRequest)
			}
			if err != nil {
				return err
			}
			if row.NodeID != node.ID {
				return fmt.Errorf("current deployment %d: belongs to a different node: %w", req.CurrentDeployment.ID, errInvalidRequest)
			}
			if row.Status == statusInProgress {
				_, err = tx.UpdateConfigDeploymentStatus(r.Context(), queries.UpdateConfigDeploymentStatusParams{
					ID:     req.CurrentDeployment.ID,
					Status: statusCompleted,
				})
				if err != nil {
					return err
				}
			}
		}

		if req.FailedDeployment != nil {
			row, err := tx.GetConfigDeploymentWithConfigVersion(r.Context(), req.FailedDeployment.ID)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed deployment %d: not found: %w", req.FailedDeployment.ID, errInvalidRequest)
			}
			if err != nil {
				return err
			}
			if row.NodeID != node.ID {
				return fmt.Errorf("failed deployment %d: belongs to a different node: %w", req.FailedDeployment.ID, errInvalidRequest)
			}
			_, err = tx.UpdateConfigDeploymentStatus(r.Context(), queries.UpdateConfigDeploymentStatusParams{
				ID:     req.FailedDeployment.ID,
				Status: statusFailed,
				Reason: nullString(req.FailedDeployment.Reason),
			})
			if err != nil {
				return err
			}
		}

		// The update channel mirrors the config transitions above. Update deployments carry node_id
		// directly, so no join is needed to check ownership.
		if req.InstallingUpdate != nil {
			dep, err := tx.GetUpdateDeployment(r.Context(), req.InstallingUpdate.ID)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("installing update %d: not found: %w", req.InstallingUpdate.ID, errInvalidRequest)
			}
			if err != nil {
				return err
			}
			if dep.NodeID != node.ID {
				return fmt.Errorf("installing update %d: belongs to a different node: %w", req.InstallingUpdate.ID, errInvalidRequest)
			}
			if dep.Status == statusPending {
				_, err = tx.SetUpdateDeploymentStatus(r.Context(), queries.SetUpdateDeploymentStatusParams{
					ID:     req.InstallingUpdate.ID,
					Status: statusInProgress,
				})
				if err != nil {
					return err
				}
			}
		}

		if req.CurrentUpdate != nil {
			dep, err := tx.GetUpdateDeployment(r.Context(), req.CurrentUpdate.ID)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("current update %d: not found: %w", req.CurrentUpdate.ID, errInvalidRequest)
			}
			if err != nil {
				return err
			}
			if dep.NodeID != node.ID {
				return fmt.Errorf("current update %d: belongs to a different node: %w", req.CurrentUpdate.ID, errInvalidRequest)
			}
			if dep.Status == statusInProgress {
				_, err = tx.SetUpdateDeploymentStatus(r.Context(), queries.SetUpdateDeploymentStatusParams{
					ID:     req.CurrentUpdate.ID,
					Status: statusCompleted,
				})
				if err != nil {
					return err
				}
			}
		}

		if req.FailedUpdate != nil {
			dep, err := tx.GetUpdateDeployment(r.Context(), req.FailedUpdate.ID)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed update %d: not found: %w", req.FailedUpdate.ID, errInvalidRequest)
			}
			if err != nil {
				return err
			}
			if dep.NodeID != node.ID {
				return fmt.Errorf("failed update %d: belongs to a different node: %w", req.FailedUpdate.ID, errInvalidRequest)
			}
			_, err = tx.SetUpdateDeploymentStatus(r.Context(), queries.SetUpdateDeploymentStatusParams{
				ID:     req.FailedUpdate.ID,
				Status: statusFailed,
				Reason: nullString(req.FailedUpdate.Reason),
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

		var currentUpdateID *int64
		if req.CurrentUpdate != nil {
			currentUpdateID = &req.CurrentUpdate.ID
		}
		var installingUpdateID *int64
		if req.InstallingUpdate != nil {
			installingUpdateID = &req.InstallingUpdate.ID
		}
		var installingUpdateError string
		var installingUpdateAttempts int
		if req.InstallingUpdate != nil {
			installingUpdateError = req.InstallingUpdate.Error
			installingUpdateAttempts = req.InstallingUpdate.Attempts
		}
		checkIn, err = tx.CreateNodeCheckIn(r.Context(), queries.CreateNodeCheckInParams{
			NodeID:                       node.ID,
			CurrentDeploymentID:          nullInt64(currentID),
			InstallingDeploymentID:       nullInt64(installingID),
			InstallingDeploymentError:    nullString(installingError),
			InstallingDeploymentAttempts: nullInt64FromInt(installingAttempts),
			CurrentUpdateDeploymentID:    nullInt64(currentUpdateID),
			InstallingUpdateDeploymentID: nullInt64(installingUpdateID),
			InstallingUpdateError:        nullString(installingUpdateError),
			InstallingUpdateAttempts:     nullInt64FromInt(installingUpdateAttempts),
		})
		if err != nil {
			return err
		}

		deployment, err = tx.GetActiveConfigDeploymentByNode(r.Context(), node.ID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			hasDeploy = false
		case err != nil:
			return err
		default:
			hasDeploy = true
			configVersion, err = tx.GetConfigVersion(r.Context(), deployment.ConfigVersionID)
			if err != nil {
				return err
			}
		}

		// The active deployment is looked up per channel (artefact kind): BOS images and supervisor RPMs
		// are independent and may both be in flight.
		activeUpdate := func(kind string) (queries.UpdateDeployment, queries.UpdateArtefact, bool, error) {
			dep, err := tx.GetActiveUpdateDeploymentByNodeAndKind(r.Context(), queries.GetActiveUpdateDeploymentByNodeAndKindParams{
				NodeID: node.ID,
				Kind:   kind,
			})
			if errors.Is(err, sql.ErrNoRows) {
				return queries.UpdateDeployment{}, queries.UpdateArtefact{}, false, nil
			}
			if err != nil {
				return queries.UpdateDeployment{}, queries.UpdateArtefact{}, false, err
			}
			art, err := tx.GetUpdateArtefact(r.Context(), dep.UpdateArtefactID)
			if err != nil {
				return queries.UpdateDeployment{}, queries.UpdateArtefact{}, false, err
			}
			return dep, art, true, nil
		}

		updateDep, updateArtefact, hasUpdate, err = activeUpdate(ArtefactKindBOSImage)
		if err != nil {
			return err
		}
		supUpdateDep, supUpdateArtefact, hasSupUpdate, err = activeUpdate(ArtefactKindSupervisorRPM)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, errUnauthorized)
			logger.Debug("check-in for unknown node")
			return
		}
		if errors.Is(err, errInvalidRequest) {
			writeError(w, errInvalidRequest)
			logger.Info("invalid check-in request", zap.Error(err))
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
			ConfigDeployment: toConfigDeployment(deployment),
			ConfigVersion:    toConfigVersion(r, configVersion),
		}
	}
	if hasUpdate {
		resp.LatestUpdate = &LatestUpdate{
			UpdateDeployment: toUpdateDeployment(updateDep),
			UpdateArtefact:   toUpdateArtefact(r, updateArtefact),
		}
	}
	if hasSupUpdate {
		resp.LatestSupervisorUpdate = &LatestUpdate{
			UpdateDeployment: toUpdateDeployment(supUpdateDep),
			UpdateArtefact:   toUpdateArtefact(r, supUpdateArtefact),
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
