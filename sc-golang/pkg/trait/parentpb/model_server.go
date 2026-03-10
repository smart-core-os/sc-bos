package parentpb

import (
	"context"
	"sort"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// ModelServer exposes Model as a traits.ParentApiServer.
type ModelServer struct {
	traits.UnimplementedParentApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) ListChildren(_ context.Context, request *traits.ListChildrenRequest) (*traits.ListChildrenResponse, error) {
	pageToken := &types.PageToken{}
	if err := decodePageToken(request.PageToken, pageToken); err != nil {
		return nil, err
	}

	lastKey := pageToken.GetLastResourceName() // the key() of the last item we sent
	pageSize := capPageSize(int(request.GetPageSize()))

	all := s.model.ListChildren()
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	nextIndex := 0
	if lastKey != "" {
		nextIndex = sort.Search(len(all), func(i int) bool {
			return all[i].Name >= lastKey
		})
		if nextIndex < len(all) && all[nextIndex].Name == lastKey {
			nextIndex++
		}
	}

	result := &traits.ListChildrenResponse{
		TotalSize: int32(len(all)),
	}
	upperBound := nextIndex + pageSize
	if upperBound > len(all) {
		upperBound = len(all)
		pageToken = nil
	} else {
		pageToken.PageStart = &types.PageToken_LastResourceName{
			LastResourceName: all[upperBound-1].Name,
		}
	}

	var err error
	result.NextPageToken, err = encodePageToken(pageToken)
	if err != nil {
		return nil, err
	}
	result.Children = all[nextIndex:upperBound]

	// apply read mask
	mask := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))
	for i, child := range result.Children {
		result.Children[i] = mask.FilterClone(child).(*traits.Child)
	}

	return result, nil
}

func (s *ModelServer) PullChildren(request *traits.PullChildrenRequest, server traits.ParentApi_PullChildrenServer) error {
	for change := range s.model.PullChildren(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&traits.PullChildrenResponse{Changes: []*traits.PullChildrenResponse_Change{change}})
		if err != nil {
			return err
		}
	}
	return nil
}
