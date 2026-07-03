// Package hubpage provides pagination helpers shared by the hub node store implementations.
package hubpage

import (
	"encoding/base64"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
)

const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

// Paginate sorts nodes by name and returns the page identified by pageSize and pageToken.
// It returns the page of nodes, a token for the next page (empty when there are no more pages),
// and the total number of nodes across all pages.
// nodes is sorted in place.
func Paginate(nodes []*hubpb.HubNode, pageSize int32, pageToken string) (page []*hubpb.HubNode, nextPageToken string, total int32, err error) {
	token := &typespb.PageToken{}
	if err := decodePageToken(pageToken, token); err != nil {
		return nil, "", 0, err
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].GetName() < nodes[j].GetName()
	})

	lastKey := token.GetLastResourceName() // the name of the last node we sent
	nextIndex := 0
	if lastKey != "" {
		nextIndex = sort.Search(len(nodes), func(i int) bool {
			return nodes[i].GetName() >= lastKey
		})
		if nextIndex < len(nodes) && nodes[nextIndex].GetName() == lastKey {
			nextIndex++
		}
	}

	upperBound := nextIndex + capPageSize(int(pageSize))
	if upperBound > len(nodes) {
		upperBound = len(nodes)
		token = nil
	} else {
		token.PageStart = &typespb.PageToken_LastResourceName{
			LastResourceName: nodes[upperBound-1].GetName(),
		}
	}

	nextPageToken, err = encodePageToken(token)
	if err != nil {
		return nil, "", 0, err
	}
	return nodes[nextIndex:upperBound], nextPageToken, int32(len(nodes)), nil
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
