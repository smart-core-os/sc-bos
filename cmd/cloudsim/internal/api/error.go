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

var (
	errInvalidRequest   = apiError{http.StatusBadRequest, "invalid_request", "invalid request"}
	errNotFound         = apiError{http.StatusNotFound, "not_found", "resource not found"}
	errForeignKey       = apiError{http.StatusBadRequest, "invalid_reference", "referenced resource does not exist"}
	errUniqueConstraint = apiError{http.StatusConflict, "conflict", "resource already exists"}
	errInternal         = apiError{http.StatusInternalServerError, "internal_error", "internal server error"}
)

// translateDBError translates database errors to predefined API errors.
func translateDBError(err error) apiError {
	if errors.Is(err, sql.ErrNoRows) {
		return errNotFound
	}
	if sqlite.IsForeignKeyError(err) {
		return errForeignKey
	}
	if sqlite.IsUniqueConstraintError(err) {
		return errUniqueConstraint
	}
	return errInternal
}

// writeError writes an error response based on the error type.
func writeError(w http.ResponseWriter, ae apiError) {
	writeJSON(w, ae.code, ErrorResponse{
		Error:   ae.errCode,
		Message: ae.message,
	})
}
