package cloud

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

// CheckInRequest is the optional request body for the check-in endpoint.
type CheckInRequest struct {
	CurrentDeployment    *CheckInDeploymentRef        `json:"currentDeployment,omitempty"`
	InstallingDeployment *CheckInInstallingDeployment `json:"installingDeployment,omitempty"`
	FailedDeployment     *CheckInFailedDeployment     `json:"failedDeployment,omitempty"`
}

// CheckInDeploymentRef references a deployment by ID.
type CheckInDeploymentRef struct {
	ID string `json:"id"`
}

// CheckInInstallingDeployment references a deployment being installed, optionally with error and attempt info.
type CheckInInstallingDeployment struct {
	ID       string `json:"id"`
	Error    string `json:"error,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
}

// CheckInFailedDeployment reports a deployment that failed.
type CheckInFailedDeployment struct {
	ID     string `json:"id"`
	Reason string `json:"reason,omitempty"`
}

// CheckInResponse is returned from the check-in endpoint.
type CheckInResponse struct {
	CheckIn      NodeCheckIn   `json:"checkIn"`
	LatestConfig *LatestConfig `json:"latestConfig,omitempty"`
}

// NodeCheckIn is the JSON representation of a node check-in.
type NodeCheckIn struct {
	NodeID      string    `json:"nodeId"`
	CheckInTime time.Time `json:"checkInTime"`
}

// LatestConfig bundles the active deployment with its config version.
type LatestConfig struct {
	Deployment    Deployment    `json:"deployment"`
	ConfigVersion ConfigVersion `json:"configVersion"`
}

// Deployment is the JSON representation of a deployment.
type Deployment struct {
	ID              string     `json:"id"`
	ConfigVersionID string     `json:"configVersionId"`
	Status          string     `json:"status"`
	StartTime       time.Time  `json:"startTime"`
	FinishedTime    *time.Time `json:"finishedTime,omitempty"`
	Reason          string     `json:"reason,omitempty"`
}

// ConfigVersion is the JSON representation of a config version.
type ConfigVersion struct {
	ID          string    `json:"id"`
	NodeID      string    `json:"nodeId"`
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

// CredentialCheckError wraps an error from a trial check-in performed to validate
// credentials before they are persisted. The server layer uses this to return a
// domain-specific status code rather than a generic internal error.
type CredentialCheckError struct{ Err error }

func (e *CredentialCheckError) Error() string { return "credential check: " + e.Err.Error() }
func (e *CredentialCheckError) Unwrap() error { return e.Err }

func IsCredentialCheckError(err error) bool {
	var e *CredentialCheckError
	return errors.As(err, &e)
}

// IsInvalidCredentialsError reports whether err indicates that the OAuth2
// client credentials were rejected by the server.
func IsInvalidCredentialsError(err error) bool {
	var re *oauth2.RetrieveError
	if errors.As(err, &re) {
		return true
	}
	var apiErr *APIError
	return errors.As(err, &apiErr) &&
		(apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden)
}

// IsConnectionError reports whether err is a network-level failure (DNS,
// connection refused, timeout, etc.) that prevented reaching the server.
func IsConnectionError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}

func IsInvalidEnrollmentCode(err error) bool {
	apiErr, ok := errors.AsType[*APIError](err)
	if !ok {
		return false
	}
	return apiErr.Code == "invalid_enrollment_code"
}
