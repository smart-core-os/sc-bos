package cloud

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// CheckInRequest is the check-in request body. A zero Running with no Progress is a valid "just
// checking in" request.
type CheckInRequest struct {
	Running  RunningState     `json:"running"`
	Progress []ProgressReport `json:"progress,omitempty"`
}

// RunningState reports what the node is currently running, plus its platform. Every field is optional.
type RunningState struct {
	Config   *RunningArtefact `json:"config,omitempty"`
	Binary   *RunningArtefact `json:"binary,omitempty"`
	Platform *Platform        `json:"platform,omitempty"`
}

// RunningArtefact identifies the version a stream is currently running. VersionID and Hash are
// best-effort drift identity and may be empty.
type RunningArtefact struct {
	Version   string `json:"version,omitempty"`
	VersionID string `json:"versionId,omitempty"`
	Hash      string `json:"hash,omitempty"`
}

// Platform is the node's runtime platform (GOOS/GOARCH).
type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// Progress states reported for an in-flight deployment.
const (
	ProgressInstalling = "installing"
	ProgressApplied    = "applied"
	ProgressFailed     = "failed"
)

// ProgressReport updates one in-flight deployment identified by its opaque DeploymentID. Error is a
// transient error (paired with installing); Reason is a terminal reason (paired with failed).
type ProgressReport struct {
	DeploymentID string `json:"deploymentId"`
	State        string `json:"state"`
	Attempts     int    `json:"attempts,omitempty"`
	Error        string `json:"error,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// CheckInResponse is returned from the check-in endpoint. LatestConfig and LatestBinary are present
// only when that stream has a non-terminal target this node has not yet reached.
type CheckInResponse struct {
	CheckIn      NodeCheckIn   `json:"checkIn"`
	LatestConfig *LatestStream `json:"latestConfig,omitempty"`
	LatestBinary *LatestStream `json:"latestBinary,omitempty"`
}

// LatestStream is the deployment a stream (config or binary) wants this node to reach, together with
// the version it targets. Its shape is identical for both streams.
type LatestStream struct {
	Deployment StreamDeployment  `json:"deployment"`
	Version    VersionProjection `json:"version"`
}

// StreamDeployment identifies the offered deployment. ID is opaque and is echoed back verbatim when
// reporting progress on it.
type StreamDeployment struct {
	ID        string `json:"id"`
	Artefact  string `json:"artefact"` // "config" or "binary"
	VersionID string `json:"versionId"`
	Status    string `json:"status"`
}

// VersionProjection is the artefact version a deployment targets: where to download it and how to
// verify it. Checksum, when set, is a type-prefixed "<algo>:<base64>" digest of the payload.
type VersionProjection struct {
	ID          string    `json:"id"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	PayloadURL  string    `json:"payloadUrl"`
	Checksum    string    `json:"checksum,omitempty"`
	CreateTime  time.Time `json:"createTime"`
}

// NodeCheckIn is the JSON representation of a node check-in.
type NodeCheckIn struct {
	NodeID      string    `json:"nodeId"`
	CheckInTime time.Time `json:"checkInTime"`
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

// IsInvalidCredentialsError reports whether err indicates that the controller's
// client certificate was rejected by the server (401/403 on an mTLS endpoint) —
// i.e. the credential no longer occupies a slot (revoked) or is otherwise
// invalid, and re-enrollment is required.
func IsInvalidCredentialsError(err error) bool {
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
