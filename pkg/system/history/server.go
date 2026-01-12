package history

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/historypb"
	"github.com/smart-core-os/sc-bos/pkg/history"
	gen_historypb "github.com/smart-core-os/sc-bos/pkg/proto/historypb"
)

type storeServer struct {
	gen_historypb.UnimplementedHistoryAdminApiServer
	store func(source string) history.Store
}

func (s *storeServer) CreateHistoryRecord(ctx context.Context, request *gen_historypb.CreateHistoryRecordRequest) (*gen_historypb.HistoryRecord, error) {
	if err := validateCreateRequest(request); err != nil {
		return nil, err
	}
	record := request.GetRecord()
	r, err := s.store(record.GetSource()).Append(ctx, record.Payload)
	if err != nil {
		return nil, err
	}
	return storeRecordToProtoRecord(record.Source, r), nil
}

func (s *storeServer) ListHistoryRecords(ctx context.Context, request *gen_historypb.ListHistoryRecordsRequest) (*gen_historypb.ListHistoryRecordsResponse, error) {
	source := request.GetQuery().GetSourceEqual()
	if source == "" {
		return nil, status.Error(codes.InvalidArgument, "source_equal must be set")
	}
	_, from := protoRecordToStoreRecord(request.GetQuery().GetFromRecord())
	_, to := protoRecordToStoreRecord(request.GetQuery().GetToRecord())

	store := s.store(source)
	pager := newPageReader(source)
	page, size, nextToken, err := pager.ListRecordsBetween(ctx, store, from, to, int(request.GetPageSize()), request.GetPageToken(), request.GetOrderBy())
	if err != nil {
		return nil, err
	}

	return &gen_historypb.ListHistoryRecordsResponse{
		Records:       page,
		NextPageToken: nextToken,
		TotalSize:     int32(size),
	}, nil
}

func newPageReader(source string) historypb.PageReader[*gen_historypb.HistoryRecord] {
	pr := historypb.NewPageReader(func(r history.Record) (*gen_historypb.HistoryRecord, error) {
		return storeRecordToProtoRecord(source, r), nil
	})
	// we use create_time in our API, so override the default record_time parsing
	pr.OrderByParser = func(s string) historypb.OrderBy {
		sn := strings.ToLower(s)
		sn = strings.Join(strings.Fields(sn), " ") // normalise whitespace
		switch sn {
		case "", "createtime", "createtime asc", "create_time", "create_time asc":
			return historypb.OrderByTimeAsc
		case "createtime desc", "create_time desc":
			return historypb.OrderByTimeDesc
		default:
			return historypb.OrderBy(s)
		}
	}
	return pr
}

func protoRecordToStoreRecord(r *gen_historypb.HistoryRecord) (string, history.Record) {
	hRecord := history.Record{
		ID:      r.GetId(),
		Payload: r.GetPayload(),
	}
	if r.GetCreateTime() != nil {
		hRecord.CreateTime = r.GetCreateTime().AsTime()
	}
	return r.GetSource(), hRecord
}

func storeRecordToProtoRecord(source string, r history.Record) *gen_historypb.HistoryRecord {
	pbRecord := &gen_historypb.HistoryRecord{
		Id:      r.ID,
		Source:  source,
		Payload: r.Payload,
	}
	if !r.CreateTime.IsZero() {
		pbRecord.CreateTime = timestamppb.New(r.CreateTime)
	}
	return pbRecord
}

func validateCreateRequest(request *gen_historypb.CreateHistoryRecordRequest) error {
	switch {
	case request.GetRecord().GetId() != "":
		return status.Error(codes.InvalidArgument, "id must not be set")
	case request.GetRecord().GetSource() == "":
		return status.Error(codes.InvalidArgument, "source must be set")
	case request.GetRecord().GetCreateTime() != nil:
		return status.Error(codes.InvalidArgument, "create_time must not be set")
	}
	return nil
}
