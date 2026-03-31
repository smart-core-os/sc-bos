package apistore

import (
	"context"
	"sync"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/proto/historypb"
)

// New creates a new history.Store backed by the given client and name.
// All records created or read by this store will have the given source.
func New(client historypb.HistoryAdminApiClient, name, source string) *Store {
	return &Store{
		slice: slice{
			client: client,
			name:   name,
			source: source,
		},
	}
}

// Store implements history.Store backed by a historypb.HistoryAdminApiClient.
// As a quirk of how the api works, it's more efficient to call Read then Len, than Len then Read.
type Store struct {
	slice
}

var _ history.Store = (*Store)(nil)

func (s *Store) Append(ctx context.Context, payload []byte) (history.Record, error) {
	pbRecord, err := s.client.CreateHistoryRecord(ctx, &historypb.CreateHistoryRecordRequest{
		Name: s.name,
		Record: &historypb.HistoryRecord{
			Source:  s.source,
			Payload: payload,
		},
	})
	if err != nil {
		return history.Record{}, err
	}
	_, hRecord := protoRecordToStoreRecord(pbRecord)
	return hRecord, nil
}

type slice struct {
	client historypb.HistoryAdminApiClient
	name   string

	source   string
	from, to history.Record

	totalSizeMu sync.Mutex
	totalSize   int
	totalSizeOk bool // true if we've successfully read totalSize from client
}

func (s *slice) Slice(from, to history.Record) history.Slice {
	return &slice{
		client: s.client,
		name:   s.name,
		source: s.source,
		from:   from,
		to:     to,
	}
}

func (s *slice) Read(ctx context.Context, into []history.Record) (int, error) {
	return s.read(ctx, into, "")
}

func (s *slice) ReadDesc(ctx context.Context, into []history.Record) (int, error) {
	return s.read(ctx, into, "create_time desc")
}

func (s *slice) read(ctx context.Context, into []history.Record, orderBy string) (int, error) {
	req := s.newListRequest(int32(len(into)))
	req.OrderBy = orderBy

	i := 0
	for {
		res, err := s.client.ListHistoryRecords(ctx, req)
		if err != nil {
			return 0, err
		}

		s.totalSizeMu.Lock()
		s.totalSize = int(res.TotalSize)
		s.totalSizeOk = true
		s.totalSizeMu.Unlock()

		for _, record := range res.Records {
			if i >= len(into) {
				// should only happen if the server doesn't respect our page size
				return i, nil
			}
			_, into[i] = protoRecordToStoreRecord(record)
			i++
		}

		req.PageToken = res.NextPageToken
		if req.PageToken == "" {
			return i, nil
		}
	}
}

func (s *slice) newListRequest(pageSize int32) *historypb.ListHistoryRecordsRequest {
	req := &historypb.ListHistoryRecordsRequest{
		Name:     s.name,
		PageSize: pageSize,
		Query: &historypb.HistoryRecord_Query{
			Source: &historypb.HistoryRecord_Query_SourceEqual{SourceEqual: s.source},
		},
	}
	if !s.from.IsZero() {
		req.Query.FromRecord = storeRecordToProtoRecord("", s.from)
	}
	if !s.to.IsZero() {
		req.Query.ToRecord = storeRecordToProtoRecord("", s.to)
	}
	return req
}

func (s *slice) Len(ctx context.Context) (int, error) {
	s.totalSizeMu.Lock()
	defer s.totalSizeMu.Unlock()
	if !s.totalSizeOk {
		req := s.newListRequest(1)
		res, err := s.client.ListHistoryRecords(ctx, req)
		if err != nil {
			return 0, err
		}
		s.totalSizeOk = true
		s.totalSize = int(res.TotalSize)
	}
	return s.totalSize, nil
}

func protoRecordToStoreRecord(r *historypb.HistoryRecord) (string, history.Record) {
	return r.GetSource(), history.Record{
		ID:         r.GetId(),
		CreateTime: r.GetCreateTime().AsTime(),
		Payload:    r.GetPayload(),
	}
}

func storeRecordToProtoRecord(source string, r history.Record) *historypb.HistoryRecord {
	return &historypb.HistoryRecord{
		Id:         r.ID,
		Source:     source,
		CreateTime: timestamppb.New(r.CreateTime),
		Payload:    r.Payload,
	}
}
