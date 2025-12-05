package historypb

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"go.uber.org/multierr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	timepb "github.com/smart-core-os/sc-api/go/types/time"
	"github.com/smart-core-os/sc-bos/pkg/history"
)

//go:generate go tool protomod protoc -- -I . -I ../../../proto --go_out=paths=source_relative:. historypb_page.proto

// NewPageReader returns a new PageReader capable of reading pages from a history.Store.
func NewPageReader[R proto.Message](decodePayload func(r history.Record) (R, error)) PageReader[R] {
	return PageReader[R]{
		DefaultPageSize: 50,
		MaxPageSize:     1000,
		DecodePayload:   decodePayload,
	}
}

// PageReader reads and decodes pages from a history.Store.
// The PageReader handles pagination, page tokens, and ordering.
type PageReader[R proto.Message] struct {
	DefaultPageSize, MaxPageSize int

	DecodePayload func(r history.Record) (R, error)
	OrderByParser func(s string) OrderBy // defaults to parseOrderBy
}

type OrderBy string

const (
	OrderByTimeAsc  OrderBy = "time asc"
	OrderByTimeDesc OrderBy = "time desc"
)

// ListRecords returns a page of records from the store between the given period.
// See ListRecordsBetween for more details.
func (pr PageReader[R]) ListRecords(ctx context.Context, store history.Store, period *timepb.Period, pageSize int, pageToken, orderBy string) (page []R, totalSize int, nextPageToken string, err error) {
	from, to := periodToRecords(period)
	return pr.ListRecordsBetween(ctx, store, from, to, pageSize, pageToken, orderBy)
}

// ListRecordsBetween returns a page of records from the store between the given from and to records.
// The page size and ordering will be honoured.
// Retrieve subsequent pages by passing a previously returned page token.
// The orderBy string must be parsable by the OrderByParser function, typically a parser for strings like "record_time asc".
func (pr PageReader[R]) ListRecordsBetween(ctx context.Context, store history.Store, from, to history.Record, pageSize int, pageToken, orderBy string) (page []R, totalSize int, nextPageToken string, err error) {
	if pageSize == 0 {
		pageSize = pr.DefaultPageSize
	}
	if pageSize > pr.MaxPageSize {
		pageSize = pr.MaxPageSize
	}

	tokenPb, err := unmarshalPageToken(pageToken)
	if err != nil {
		return nil, 0, "", status.Error(codes.InvalidArgument, "invalid page token")
	}

	slice := store.Slice(from, to)
	totalSize = int(tokenPb.TotalSize)
	if totalSize == 0 {
		// avoid this potentially expensive step if possible
		totalSize, err = slice.Len(ctx)
		if err != nil {
			return nil, 0, "", err
		}
	}

	parseOrderBy := parseOrderBy
	if pr.OrderByParser != nil {
		parseOrderBy = pr.OrderByParser
	}
	reader, pager, nexter, err := orderByFuncs(parseOrderBy(orderBy))
	if err != nil {
		return nil, 0, "", err
	}

	if tokenPb.RecordId != "" {
		slice = pager(slice, from, to, history.Record{ID: tokenPb.RecordId})
	}

	dst := make([]history.Record, pageSize+1) // +1 to know if there's a next page or not
	read, err := reader(slice, ctx, dst)
	if err != nil {
		return nil, 0, "", err
	}

	readRecords := read
	if readRecords > pageSize {
		readRecords = pageSize
	}
	page = make([]R, readRecords)

	var allErrs error
	for i := range readRecords {
		r, err := pr.DecodePayload(dst[i])
		if err != nil {
			allErrs = multierr.Append(allErrs, fmt.Errorf("%s %w", dst[i].ID, err))
			continue
		}
		page[i] = r
	}

	if read <= pageSize {
		return
	}

	nextPageToken, err = marshalPageToken(&PageToken{RecordId: nexter(dst).ID, TotalSize: int32(totalSize)})
	return
}

func periodToRecords(p *timepb.Period) (from, to history.Record) {
	if p == nil {
		return
	}
	if p.StartTime != nil {
		from.CreateTime = p.StartTime.AsTime()
	}
	if p.EndTime != nil {
		to.CreateTime = p.EndTime.AsTime()
	}
	return
}

func unmarshalPageToken(token string) (*PageToken, error) {
	if token == "" {
		return &PageToken{}, nil
	}
	data, err := base64.RawStdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	pb := &PageToken{}
	err = proto.Unmarshal(data, pb)
	return pb, err
}

func marshalPageToken(pb *PageToken) (string, error) {
	data, err := proto.Marshal(pb)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(data), nil
}

// These encapsulate the differences between ascending and descending order.
// sliceReadFunc calls Slice.Read or Slice.ReadDesc.
// slicePagerFunc applies page tokens to the slice, either slice[:i] or slice[i:].
// sliceNextFunc returns the record that will be used as the next page token, closely related to slicePagerFunc.
type (
	sliceReadFunc  = func(slice history.Slice, ctx context.Context, dst []history.Record) (int, error)
	slicePagerFunc = func(slice history.Slice, from, to, token history.Record) history.Slice
	sliceNextFunc  = func(all []history.Record) history.Record
)

func parseOrderBy(s string) OrderBy {
	sn := strings.ToLower(s)
	sn = strings.Join(strings.Fields(sn), " ") // normalize spaces
	switch sn {
	case "", "recordtime", "recordtime asc", "record_time", "record_time asc":
		return OrderByTimeAsc
	case "recordtime desc", "record_time desc":
		return OrderByTimeDesc
	default:
		return OrderBy(s)
	}
}

func orderByFuncs(orderBy OrderBy) (sliceReadFunc, slicePagerFunc, sliceNextFunc, error) {
	switch orderBy {
	case OrderByTimeAsc:
		return history.Slice.Read, ascPager, ascNext, nil
	case OrderByTimeDesc:
		return history.Slice.ReadDesc, descPager, descNext, nil
	default:
		return nil, nil, nil, status.Errorf(codes.InvalidArgument, "invalid order by %q", orderBy)
	}
}

func ascPager(slice history.Slice, _, to, token history.Record) history.Slice {
	return slice.Slice(token, to)
}

func descPager(slice history.Slice, from, _, token history.Record) history.Slice {
	return slice.Slice(from, token)
}

func ascNext(all []history.Record) history.Record {
	if len(all) < 1 {
		return history.Record{}
	}
	// The first record of the next page should be the last record read, we read +1 for this purpose
	return all[len(all)-1]
}

func descNext(all []history.Record) history.Record {
	if len(all) < 2 {
		return history.Record{}
	}
	// The first record of the next page should be the last record we return as it will be used as the endTime which
	// is exclusive. -2 because we've already read 1 past the end of the returned items to see if there will be more
	// pages.
	return all[len(all)-2]
}
