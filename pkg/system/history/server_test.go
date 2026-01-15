package history

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/history/memstore"
	"github.com/smart-core-os/sc-bos/pkg/proto/historypb"
)

func Test_storeServer_ListHistoryRecords(t *testing.T) {
	now := time.Unix(0, 0)
	store := memstore.New(memstore.WithNow(func() time.Time {
		return now
	}))
	server := &storeServer{
		store: func(source string) history.Store {
			return store
		},
	}

	// create 6 records, each 10s apart, starting a minute ago: [0s, 10s, ..., 50s]
	records := make([]history.Record, 6)
	for i := range 6 {
		secs := int64(i * 10)
		now = time.Unix(secs, 0)
		var err error
		records[i], err = store.Append(nil, []byte(fmt.Sprintf("%ds", secs)))
		if err != nil {
			t.Fatalf("failed to append record: %v", err)
		}
	}
	now = time.Unix(60, 0)

	tests := []struct {
		req  *historypb.ListHistoryRecordsRequest
		want [][]history.Record
	}{
		{req: reqTo(2, "", 60), want: pages(records[0:2], records[2:4], records[4:6])},
		{req: reqFrom(2, "", 10), want: pages(records[1:3], records[3:5], records[5:6])},
		{req: reqBetween(2, "", 10, 50), want: pages(records[1:3], records[3:5])},
		{req: reqTo(2, "create_time desc", 60), want: pages(r(records[4:6]), r(records[2:4]), r(records[0:2]))},
		{req: reqFrom(2, "create_time desc", 10), want: pages(r(records[4:6]), r(records[2:4]), r(records[1:2]))},
		{req: reqBetween(2, "create_time desc", 10, 50), want: pages(r(records[3:5]), r(records[1:3]))},
	}

	for _, tt := range tests {
		t.Run(reqToName(tt.req), func(t *testing.T) {
			want := tt.want
			var wantTotalSize int32
			for _, page := range want {
				wantTotalSize += int32(len(page))
			}
			req := proto.Clone(tt.req).(*historypb.ListHistoryRecordsRequest)
			var pageNum int
			for {
				res, err := server.ListHistoryRecords(context.Background(), req)
				if err != nil {
					t.Fatalf("ListHistoryRecords(%v) = %v", pageNum, err)
				}
				if res.TotalSize != wantTotalSize {
					t.Errorf("ListHistoryRecords(%v) = %d, want %d", pageNum, res.TotalSize, wantTotalSize)
				}
				if diff := cmp.Diff(modelToProto(want[0]), res.Records, protocmp.Transform()); diff != "" {
					t.Errorf("ListHistoryRecords(%v) records (-want +got):\n%s", pageNum, diff)
				}

				// prepare for the next page
				pageNum++
				want = want[1:]
				req.PageToken = res.NextPageToken
				if req.PageToken == "" {
					if len(want) > 0 {
						t.Errorf("unexpected absent next page token, want %d more pages", len(want))
					}
					break
				}
				if len(want) == 0 {
					t.Errorf("unexpected next page token, want no more pages")
					break
				}
			}
		})
	}
}

func reqToName(req *historypb.ListHistoryRecordsRequest) string {
	q := req.GetQuery()
	period := make([]string, 2)
	if q.FromRecord != nil {
		period[0] = fmt.Sprintf("%ds", q.FromRecord.CreateTime.Seconds)
	}
	if q.ToRecord != nil {
		period[1] = fmt.Sprintf("%ds", q.ToRecord.CreateTime.Seconds)
	}
	return fmt.Sprintf("[%v) %v", strings.Join(period, ","), req.OrderBy)
}

func reqBetween(pageSize int32, orderBy string, from, to int64) *historypb.ListHistoryRecordsRequest {
	return &historypb.ListHistoryRecordsRequest{
		Query: &historypb.HistoryRecord_Query{
			Source: &historypb.HistoryRecord_Query_SourceEqual{SourceEqual: "test"},
			FromRecord: &historypb.HistoryRecord{
				CreateTime: timestamppb.New(time.Unix(from, 0)),
			},
			ToRecord: &historypb.HistoryRecord{
				CreateTime: timestamppb.New(time.Unix(to, 0)),
			},
		},
		PageSize: pageSize,
		OrderBy:  orderBy,
	}
}

func reqFrom(pageSize int32, orderBy string, from int64) *historypb.ListHistoryRecordsRequest {
	return &historypb.ListHistoryRecordsRequest{
		Query: &historypb.HistoryRecord_Query{
			Source: &historypb.HistoryRecord_Query_SourceEqual{SourceEqual: "test"},
			FromRecord: &historypb.HistoryRecord{
				CreateTime: timestamppb.New(time.Unix(from, 0)),
			},
		},
		PageSize: pageSize,
		OrderBy:  orderBy,
	}
}

func reqTo(pageSize int32, orderBy string, to int64) *historypb.ListHistoryRecordsRequest {
	return &historypb.ListHistoryRecordsRequest{
		Query: &historypb.HistoryRecord_Query{
			Source: &historypb.HistoryRecord_Query_SourceEqual{SourceEqual: "test"},
			ToRecord: &historypb.HistoryRecord{
				CreateTime: timestamppb.New(time.Unix(to, 0)),
			},
		},
		PageSize: pageSize,
		OrderBy:  orderBy,
	}
}

func pages(pages ...[]history.Record) [][]history.Record {
	return pages
}

func r(records []history.Record) []history.Record {
	c := make([]history.Record, len(records))
	copy(c, records)
	slices.Reverse(c)
	return c
}

func modelToProto(records []history.Record) []*historypb.HistoryRecord {
	models := make([]*historypb.HistoryRecord, len(records))
	for i, r := range records {
		models[i] = storeRecordToProtoRecord("test", r)
	}
	return models
}
