// Package page provides utilities for paging through lists of items.
// The items this package work with are usually already in a slice in memory.
package page

import (
	"encoding/base64"
	"slices"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/sc-api/go/types"
)

type ListRequest interface {
	GetPageToken() string
	GetPageSize() int32
}

// List returns a page of items from the return value of the list function.
// The idFunc should return a unique, and stable, identifier for each item.
func List[T any](req ListRequest, idFunc func(T) string, list func() []T, opts ...ListOption) (_ []T, totalSize int, nextPageToken string, _ error) {
	lo := newListOptions(opts...)

	pageToken := &types.PageToken{}
	if err := decodePageToken(req.GetPageToken(), pageToken); err != nil {
		return nil, 0, "", err
	}

	lastName := pageToken.GetLastResourceName() // The name of the last item of the previous page
	pageSize := lo.capPageSize(req.GetPageSize())

	all := list()
	slices.SortFunc(all, func(a, b T) int {
		return strings.Compare(idFunc(a), idFunc(b))
	})

	nextIndex := 0
	if lastName != "" {
		var found bool
		nextIndex, found = slices.BinarySearchFunc(all, lastName, func(t T, s string) int {
			return strings.Compare(idFunc(t), s)
		})
		if found {
			nextIndex++
		}
	}

	endOfPage := nextIndex + pageSize
	if endOfPage < len(all) {
		pageToken.PageStart = &types.PageToken_LastResourceName{
			LastResourceName: idFunc(all[endOfPage-1]),
		}
	} else {
		// no more pages
		endOfPage = len(all)
		pageToken = nil
	}

	nextPageToken, err := encodePageToken(pageToken)
	if err != nil {
		return nil, 0, "", err
	}

	return all[nextIndex:endOfPage], len(all), nextPageToken, nil
}

type listOptions struct {
	defaultPageSize int
	maxPageSize     int
}

func newListOptions(opts ...ListOption) listOptions {
	lo := defaultListOptions // copy
	for _, opt := range opts {
		opt(&lo)
	}
	return lo
}

var defaultListOptions = listOptions{
	defaultPageSize: 50,
	maxPageSize:     1000,
}

func (lo listOptions) capPageSize(pageSize int32) int {
	if pageSize == 0 {
		return lo.defaultPageSize
	}
	if int(pageSize) > lo.maxPageSize {
		return lo.maxPageSize
	}
	return int(pageSize)
}

type ListOption func(o *listOptions)

// WithDefaultPageSize sets the default page size for the list operation.
func WithDefaultPageSize(size int) ListOption {
	return func(o *listOptions) {
		o.defaultPageSize = size
	}
}

// WithMaxPageSize sets the maximum page size for the list operation.
func WithMaxPageSize(size int) ListOption {
	return func(o *listOptions) {
		o.maxPageSize = size
	}
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
