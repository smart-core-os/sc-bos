package healthhistory

import (
	"context"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/internal/health/healthdb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/historypb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// Server is a [healthpb.HealthHistoryServer] that reads health check history from a database.
type Server struct {
	healthpb.UnimplementedHealthHistoryServer
	db ServerStore
}

// A ServerStore provides access to health check history records.
type ServerStore interface {
	Read(ctx context.Context, id healthdb.CheckID, from, to healthdb.RecordID, desc bool, dst []healthdb.Record) (n int, err error)
	Count(ctx context.Context, id healthdb.CheckID, from, to healthdb.RecordID) (total int, err error)
}

func NewServer(db ServerStore) *Server {
	return &Server{db: db}
}

func (s *Server) ListHealthCheckHistory(ctx context.Context, req *healthpb.ListHealthCheckHistoryRequest) (*healthpb.ListHealthCheckHistoryResponse, error) {
	id := parseCheckID(req)
	from, to := parseListBounds(req)
	orderBy := parseOrderBy(req.GetOrderBy())
	nextID, pageSize, totalSize, err := parsePageInfo(req)
	if err != nil {
		return nil, err
	}

	// avoid counting if we already know the total size
	if totalSize == 0 {
		t, err := s.db.Count(ctx, id, from, to)
		if err != nil {
			return nil, err
		}
		totalSize = int32(t)
	}

	if nextID != 0 {
		switch orderBy {
		case historypb.OrderByTimeAsc:
			from = nextID
		case historypb.OrderByTimeDesc:
			to = nextID + 1 // +1 because to is exclusive
		}
	}

	buf := make([]healthdb.Record, pageSize+1) // +1 to detect if there's a next page
	n, err := s.db.Read(ctx, id, from, to, orderBy == historypb.OrderByTimeDesc, buf)
	if err != nil {
		return nil, err
	}

	hasMorePages := n == len(buf)
	res := &healthpb.ListHealthCheckHistoryResponse{
		TotalSize: totalSize,
	}
	if hasMorePages {
		token, err := createNextPageToken(buf[n-1], totalSize)
		if err != nil {
			return nil, err
		}
		res.NextPageToken = token
		n-- // don't include the extra record in the results
	}
	res.HealthCheckRecords = make([]*healthpb.HealthCheckRecord, n)
	for i := range n {
		hcr, err := decodeRecord(buf[i])
		if err != nil {
			return nil, err
		}
		res.HealthCheckRecords[i] = hcr
	}
	return res, nil
}

func parseCheckID(req *healthpb.ListHealthCheckHistoryRequest) healthdb.CheckID {
	return healthdb.CheckID{Name: req.GetName(), ID: req.GetId()}
}

func parseListBounds(req *healthpb.ListHealthCheckHistoryRequest) (from, to healthdb.RecordID) {
	if ts := req.GetPeriod().GetStartTime(); ts != nil {
		from = healthdb.MakeRecordID(ts.AsTime(), 0)
	}
	if ts := req.GetPeriod().GetEndTime(); ts != nil {
		to = healthdb.MakeRecordID(ts.AsTime(), 0)
	}
	return from, to
}

func parseOrderBy(s string) historypb.OrderBy {
	sn := strings.ToLower(s)
	sn = strings.Join(strings.Fields(sn), " ") // normalize spaces
	switch sn {
	case "", "recordtime", "recordtime asc", "record_time", "record_time asc":
		return historypb.OrderByTimeAsc
	case "recordtime desc", "record_time desc":
		return historypb.OrderByTimeDesc
	default:
		return historypb.OrderBy(s)
	}
}

func decodeRecord(r healthdb.Record) (*healthpb.HealthCheckRecord, error) {
	dst := &healthpb.HealthCheckRecord{
		RecordTime: timestamppb.New(r.ID.Timestamp()),
		HealthCheck: &healthpb.HealthCheck{
			Id: r.CheckID,
		},
	}
	opts := proto.UnmarshalOptions{
		Merge:        true,
		AllowPartial: true,
	}
	if err := opts.Unmarshal(r.Aux, dst.HealthCheck); err != nil {
		return nil, err
	}
	if err := opts.Unmarshal(r.Main, dst.HealthCheck); err != nil {
		return nil, err
	}
	return dst, nil
}
