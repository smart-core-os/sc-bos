package sim

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/smart-core-os/sc-bos/internal/sqlite"
)

// ErrorResponse is the JSON structure returned for API errors.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// apiError represents an API error with HTTP status code and error details.
type apiError struct {
	code    int
	errCode string
	message string
}

func (e apiError) Error() string {
	return e.message
}

func (e apiError) internal() bool {
	return e.code >= 500
}

var (
	errInvalidRequest             = apiError{http.StatusBadRequest, "invalid_request", "invalid request"}
	errNotFound                   = apiError{http.StatusNotFound, "not_found", "resource not found"}
	errForeignKey                 = apiError{http.StatusBadRequest, "invalid_reference", "referenced resource does not exist"}
	errUniqueConstraint           = apiError{http.StatusConflict, "conflict", "resource already exists"}
	errUnauthorized               = apiError{http.StatusUnauthorized, "unauthorized", "invalid or missing authentication"}
	errInternal                   = apiError{http.StatusInternalServerError, "internal_error", "internal server error"}
	errConfigDeploymentInProgress = apiError{http.StatusConflict, "deployment_in_progress", "a deployment is already in progress for this node"}
	errBinaryDeploymentInProgress = apiError{http.StatusConflict, "binary_in_progress", "a binary is already in progress for this node"}
)

func errChangePlatform(old, new Platform) apiError {
	return apiError{
		code:    http.StatusConflict,
		errCode: "platform_mismatch",
		message: fmt.Sprintf("node attempted to change platform from {%q,%q} to {%q,%q}", old.OS, old.Arch, new.OS, new.Arch),
	}
}

// translateDBError translates database errors to predefined API errors.
func translateDBError(err error) apiError {
	if passthrough, ok := errors.AsType[apiError](err); ok {
		return passthrough
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return errNotFound
	case sqlite.IsForeignKeyError(err):
		return errForeignKey
	case sqlite.IsUniqueConstraintError(err):
		return errUniqueConstraint
	default:
		return errInternal
	}
}

// writeError writes an error response based on the error type.
func writeError(w http.ResponseWriter, ae apiError) {
	writeJSON(w, ae.code, ErrorResponse{
		Error:   ae.errCode,
		Message: ae.message,
	})
}
