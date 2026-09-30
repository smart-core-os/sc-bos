package accesscredentialpb

import (
	"cmp"
	"encoding/base64"
	"slices"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
)

const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

// ListPage returns one page of all, as requested by req.
// Credentials are ordered by CompareIDs and paged by id, so credentials added or removed between pages don't cause
// others to be skipped or repeated.
// all is sorted in place and must not be modified by the caller afterwards, returned credentials are clones.
func ListPage(all []*Credential, req *ListCredentialsRequest) (*ListCredentialsResponse, error) {
	filter := masks.NewResponseFilter(masks.WithFieldMask(req.GetReadMask()))
	// FilterClone panics on a path it can't follow, e.g. into the more map
	if err := filter.Validate(&Credential{}); err != nil {
		return nil, err
	}
	pageToken := &typespb.PageToken{}
	if err := decodePageToken(req.GetPageToken(), pageToken); err != nil {
		return nil, err
	}
	pageSize := capPageSize(int(req.GetPageSize()))

	slices.SortFunc(all, func(a, b *Credential) int {
		return CompareIDs(a.GetId(), b.GetId())
	})

	nextIndex := 0
	if lastID := pageToken.GetLastResourceName(); lastID != "" {
		nextIndex, _ = slices.BinarySearchFunc(all, lastID, func(c *Credential, id string) int {
			return CompareIDs(c.GetId(), id)
		})
		if nextIndex < len(all) && all[nextIndex].GetId() == lastID {
			nextIndex++
		}
	}

	res := &ListCredentialsResponse{TotalSize: int32(len(all))}
	upperBound := nextIndex + pageSize
	if upperBound >= len(all) {
		upperBound = len(all)
	} else {
		var err error
		res.NextPageToken, err = encodePageToken(&typespb.PageToken{
			PageStart: &typespb.PageToken_LastResourceName{LastResourceName: all[upperBound-1].GetId()},
		})
		if err != nil {
			return nil, err
		}
	}

	for _, c := range all[nextIndex:upperBound] {
		res.Credentials = append(res.Credentials, filter.FilterClone(c).(*Credential))
	}
	return res, nil
}

// CompareIDs orders credential ids.
// Numeric ids are ordered numerically and sort before non-numeric ids, which are ordered lexically.
func CompareIDs(a, b string) int {
	an, aErr := strconv.ParseInt(a, 10, 64)
	bn, bErr := strconv.ParseInt(b, 10, 64)
	switch {
	case aErr == nil && bErr == nil:
		return cmp.Compare(an, bn)
	case aErr == nil:
		return -1
	case bErr == nil:
		return 1
	default:
		return cmp.Compare(a, b)
	}
}

func capPageSize(pageSize int) int {
	if pageSize <= 0 {
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
	tokenBytes, err := proto.Marshal(pageToken)
	if err != nil {
		return "", status.Errorf(codes.Unknown, "unable to create page token: %v", err)
	}
	return base64.StdEncoding.EncodeToString(tokenBytes), nil
}
