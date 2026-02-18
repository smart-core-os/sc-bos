package api

import (
	"database/sql"
	"errors"
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
	errInvalidRequest   = apiError{http.StatusBadRequest, "invalid_request", "invalid request"}
	errNotFound         = apiError{http.StatusNotFound, "not_found", "resource not found"}
	errForeignKey       = apiError{http.StatusBadRequest, "invalid_reference", "referenced resource does not exist"}
	errUniqueConstraint = apiError{http.StatusConflict, "conflict", "resource already exists"}
	errUnauthorized     = apiError{http.StatusUnauthorized, "unauthorized", "invalid or missing authentication"}
	errInternal         = apiError{http.StatusInternalServerError, "internal_error", "internal server error"}
)

// translateDBError translates database errors to predefined API errors.
func translateDBError(err error) apiError {
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
