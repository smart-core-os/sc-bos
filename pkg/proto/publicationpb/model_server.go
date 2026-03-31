package publicationpb

import (
	"context"
	"sort"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

var VersionMismatchErr = status.Error(codes.FailedPrecondition, "version mismatch: update version != server version")

type ModelServer struct {
	UnimplementedPublicationApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterPublicationApiServer(server, m)
}

func (m *ModelServer) CreatePublication(_ context.Context, request *CreatePublicationRequest) (*Publication, error) {
	return m.model.CreatePublication(request.Publication, WithNewVersion(), WithNewPublishTime(), WithResetReceipt())
}

func (m *ModelServer) GetPublication(_ context.Context, request *GetPublicationRequest) (*Publication, error) {
	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	publication, ok := m.model.GetPublication(request.Id, resource.WithReadMask(request.ReadMask))
	if !ok {
		return nil, status.Errorf(codes.NotFound, "publication.id %s", request.Id)
	}
	return publication, nil
}

func (m *ModelServer) UpdatePublication(_ context.Context, request *UpdatePublicationRequest) (*Publication, error) {
	if request.Publication.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	return m.model.UpdatePublication(request.Publication.Id, request.Publication,
		WithNewVersion(), WithNewPublishTime(), WithResetReceipt(),
		resource.WithUpdateMask(request.UpdateMask),
		resource.WithExpectedCheck(func(msg proto.Message) error {
			if request.Version == "" {
				return nil
			}
			val := msg.(*Publication)
			if val.Version == request.Version {
				return nil
			}
			return VersionMismatchErr
		}),
	)
}

func (m *ModelServer) DeletePublication(_ context.Context, request *DeletePublicationRequest) (*Publication, error) {
	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	return m.model.DeletePublication(request.Id, resource.WithAllowMissing(request.AllowMissing), resource.WithExpectedCheck(func(msg proto.Message) error {
		val := msg.(*Publication)
		if request.Version != "" && val.Version != request.Version {
			return VersionMismatchErr
		}

		return nil
	}))
}

func (m *ModelServer) PullPublication(request *PullPublicationRequest, server PublicationApi_PullPublicationServer) error {
	if request.Id == "" {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	for change := range m.model.PullPublication(server.Context(), request.Id, resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullPublicationResponse{Changes: []*PullPublicationResponse_Change{
			{Name: request.Name, ChangeTime: timestamppb.New(change.ChangeTime), Publication: change.Value},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ModelServer) ListPublications(_ context.Context, request *ListPublicationsRequest) (*ListPublicationsResponse, error) {
	pageToken := &typespb.PageToken{}
	if err := decodePageToken(request.PageToken, pageToken); err != nil {
		return nil, err
	}

	lastKey := pageToken.GetLastResourceName() // the key() of the last item we sent
	pageSize := capPageSize(int(request.GetPageSize()))

	sortedItems := m.model.ListPublications(resource.WithReadMask(request.ReadMask))
	nextIndex := 0
	if lastKey != "" {
		nextIndex = sort.Search(len(sortedItems), func(i int) bool {
			return sortedItems[i].Id > lastKey
		})
	}

	result := &ListPublicationsResponse{
		TotalSize: int32(len(sortedItems)),
	}
	upperBound := nextIndex + pageSize
	if upperBound > len(sortedItems) {
		upperBound = len(sortedItems)
		pageToken = nil
	} else {
		pageToken.PageStart = &typespb.PageToken_LastResourceName{
			LastResourceName: sortedItems[upperBound-1].Id,
		}
	}

	var err error
	result.NextPageToken, err = encodePageToken(pageToken)
	if err != nil {
		return nil, err
	}
	result.Publications = sortedItems[nextIndex:upperBound]
	return result, nil
}

func (m *ModelServer) PullPublications(request *PullPublicationsRequest, server PublicationApi_PullPublicationsServer) error {
	for change := range m.model.PullPublications(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullPublicationsResponse{Changes: []*PullPublicationsResponse_Change{
			{Name: request.Name, Type: change.ChangeType, ChangeTime: timestamppb.New(change.ChangeTime), OldValue: change.OldValue, NewValue: change.NewValue},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ModelServer) AcknowledgePublication(_ context.Context, request *AcknowledgePublicationRequest) (*Publication, error) {
	if request.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if request.Version == "" {
		return nil, status.Error(codes.InvalidArgument, "version is required")
	}
	received := &Publication{Audience: &Publication_Audience{
		Receipt:               request.Receipt,
		ReceiptRejectedReason: request.ReceiptRejectedReason,
	}}

	// we return this error if the publication has already been ack'd,
	// we check for it later in combination with request.AllowAcknowledged
	alreadyAcknowledged := status.Errorf(codes.FailedPrecondition, "%v@%v is already acknowledged", request.Id, request.Version)
	var acknowledgedPub *Publication

	p, err := m.model.UpdatePublication(request.Id, received,
		resource.WithUpdatePaths("audience.receipt", "audience.receipt_rejected_reason"),
		resource.WithExpectedCheck(func(msg proto.Message) error {
			val := msg.(*Publication)
			// check the version matches
			if val.Version != request.Version {
				return status.Error(codes.Aborted, "version mismatch: acknowledge version != server version")
			}
			// check an ACK hasn't already been processed
			receipt := val.GetAudience().GetReceipt()
			if receipt == Publication_Audience_ACCEPTED || receipt == Publication_Audience_REJECTED {
				acknowledgedPub = val
				return alreadyAcknowledged
			}

			return nil
		}),
		resource.InterceptAfter(func(old, new proto.Message) {
			newVal := new.(*Publication)
			newVal.Audience.ReceiptTime = timestamppb.New(m.model.publications.Clock().Now())
		}),
	)

	if err == alreadyAcknowledged && request.AllowAcknowledged {
		return acknowledgedPub, nil
	}

	return p, err
}
