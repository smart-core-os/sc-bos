package hailpb

import (
	"encoding/base64"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/types"
)

const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

func capPageSize(pageSize int) int {
	if pageSize == 0 {
		return defaultPageSize
	}
	if pageSize > maxPageSize {
		return maxPageSize
	}
	return pageSize
}

func decodePageToken(token string, pageToken *types.PageToken) error {
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

func encodePageToken(pageToken *types.PageToken) (string, error) {
	if pageToken != nil {
		tokenBytes, err := proto.Marshal(pageToken)
		if err != nil {
			return "", status.Errorf(codes.Unknown, "unable to create page token: %v", err)
		}
		return base64.StdEncoding.EncodeToString(tokenBytes), nil
	}
	return "", nil
}
