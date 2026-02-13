package api

import (
	"net/http"
	"strconv"
)

const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

// ListResponse wraps a list of items with pagination info.
type ListResponse[T any] struct {
	Items         []T    `json:"items"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

// parsePagination extracts pagination parameters from the request.
// recognized URL parameters:
//   - pageToken - if present, should be a valid page token. If absent, afterID = 0
//   - pageSize - if present, must be a positive integer. Clamped to [1, maxPageSize]. If absent, limit = defaultPageSize
//
// Returns afterID (decoded from pageToken), limit, and any error.
// If parameters are present but invalid, returns errInvalidRequest.
func parsePagination(r *http.Request) (afterID int64, limit int64, err error) {
	// Parse page token (base36-encoded ID)
	if token := r.URL.Query().Get("pageToken"); token != "" {
		var ok bool
		afterID, ok = decodePageToken(token)
		if !ok || afterID < 0 {
			return 0, 0, errInvalidRequest
		}
	}

	// Parse page size
	limit = defaultPageSize
	if sizeStr := r.URL.Query().Get("pageSize"); sizeStr != "" {
		size, err := strconv.ParseInt(sizeStr, 10, 64)
		if err != nil || size < 1 {
			return 0, 0, errInvalidRequest
		}
		limit = size
		if limit > maxPageSize {
			limit = maxPageSize
		}
	}

	return afterID, limit, nil
}

// encodePageToken encodes an ID as a page token.
// we just use a plain base 10 integer for this simulator, because that makes human interpretation of values easier
// On a real implementation, you'd perhaps want a different ID and page token format.
func encodePageToken(id int64) string {
	return strconv.FormatInt(id, 10)
}

func decodePageToken(pageToken string) (afterID int64, ok bool) {
	afterID, err := strconv.ParseInt(pageToken, 10, 64)
	return afterID, err == nil
}
