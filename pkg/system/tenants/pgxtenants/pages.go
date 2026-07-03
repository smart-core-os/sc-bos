package pgxtenants

import (
	"encoding/base64"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
)

const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

// paginate sorts items by key and returns the page identified by pageSize and pageToken.
// It returns the page, a token for the next page (empty when there are no more pages),
// and the total number of items across all pages. items is sorted in place.
func paginate[T any](items []T, pageSize int32, pageToken string, key func(T) string) (page []T, nextPageToken string, total int32, err error) {
	token := &typespb.PageToken{}
	if err := decodePageToken(pageToken, token); err != nil {
		return nil, "", 0, err
	}

	sort.Slice(items, func(i, j int) bool {
		return key(items[i]) < key(items[j])
	})

	lastKey := token.GetLastResourceName() // the id of the last item we sent
	nextIndex := 0
	if lastKey != "" {
		nextIndex = sort.Search(len(items), func(i int) bool {
			return key(items[i]) >= lastKey
		})
		if nextIndex < len(items) && key(items[nextIndex]) == lastKey {
			nextIndex++
		}
	}

	upperBound := nextIndex + capPageSize(int(pageSize))
	if upperBound > len(items) {
		upperBound = len(items)
		token = nil
	} else {
		token.PageStart = &typespb.PageToken_LastResourceName{
			LastResourceName: key(items[upperBound-1]),
		}
	}

	nextPageToken, err = encodePageToken(token)
	if err != nil {
		return nil, "", 0, err
	}
	return items[nextIndex:upperBound], nextPageToken, int32(len(items)), nil
}

func capPageSize(pageSize int) int {
	if pageSize == 0 {
		return defaultPageSize
	}
	if pageSize > maxPageSize {
		return maxPageSize
	}
	return pageSize
}

func decodePageToken(token string, pageToken *typespb.PageToken) error {
	if token != "" {
		tokenBytes, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "bad page token: %v", err)
		}
		if err := proto.Unmarshal(tokenBytes, pageToken); err != nil {
			return status.Errorf(codes.InvalidArgument, "bad page token: %v", err)
		}
	}
	return nil
}

func encodePageToken(pageToken *typespb.PageToken) (string, error) {
	if pageToken != nil {
		tokenBytes, err := proto.Marshal(pageToken)
		if err != nil {
			return "", status.Errorf(codes.Unknown, "unable to create page token: %v", err)
		}
		return base64.StdEncoding.EncodeToString(tokenBytes), nil
	}
	return "", nil
}
