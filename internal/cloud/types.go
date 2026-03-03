package cloud

import (
	"fmt"
	"time"
)

// CheckInRequest is the optional request body for the check-in endpoint.
type CheckInRequest struct {
	CurrentDeployment    *CheckInDeploymentRef        `json:"currentDeployment,omitempty"`
	InstallingDeployment *CheckInInstallingDeployment `json:"installingDeployment,omitempty"`
	FailedDeployment     *CheckInFailedDeployment     `json:"failedDeployment,omitempty"`
}

// CheckInDeploymentRef references a deployment by ID.
type CheckInDeploymentRef struct {
	ID int64 `json:"id"`
}

// CheckInInstallingDeployment references a deployment being installed, optionally with error and attempt info.
type CheckInInstallingDeployment struct {
	ID       int64  `json:"id"`
	Error    string `json:"error,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
}

// CheckInFailedDeployment reports a deployment that failed.
type CheckInFailedDeployment struct {
	ID     int64  `json:"id"`
	Reason string `json:"reason,omitempty"`
}

// CheckInResponse is returned from the check-in endpoint.
type CheckInResponse struct {
	CheckIn      NodeCheckIn   `json:"checkIn"`
	LatestConfig *LatestConfig `json:"latestConfig,omitempty"`
}

// NodeCheckIn is the JSON representation of a node check-in.
type NodeCheckIn struct {
	ID                           int64     `json:"id"`
	NodeID                       int64     `json:"nodeId"`
	CheckInTime                  time.Time `json:"checkInTime"`
	CurrentDeploymentID          *int64    `json:"currentDeploymentId,omitempty"`
	InstallingDeploymentID       *int64    `json:"installingDeploymentId,omitempty"`
	InstallingDeploymentError    string    `json:"installingDeploymentError,omitempty"`
	InstallingDeploymentAttempts *int64    `json:"installingDeploymentAttempts,omitempty"`
}

// LatestConfig bundles the active deployment with its config version.
type LatestConfig struct {
	Deployment    Deployment    `json:"deployment"`
	ConfigVersion ConfigVersion `json:"configVersion"`
}

// Deployment is the JSON representation of a deployment.
type Deployment struct {
	ID              int64      `json:"id"`
	ConfigVersionID int64      `json:"configVersionId"`
	Status          string     `json:"status"`
	StartTime       time.Time  `json:"startTime"`
	FinishedTime    *time.Time `json:"finishedTime,omitempty"`
	Reason          string     `json:"reason,omitempty"`
}

// ConfigVersion is the JSON representation of a config version.
type ConfigVersion struct {
	ID          int64     `json:"id"`
	NodeID      int64     `json:"nodeId"`
	Description string    `json:"description"`
	PayloadURL  string    `json:"payloadUrl"`
	CreateTime  time.Time `json:"createTime"`
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"error"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("HTTP %d: %s: %s", e.StatusCode, e.Code, e.Message)
}
