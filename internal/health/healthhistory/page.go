package healthhistory

import (
	"encoding/base64"

	"google.golang.org/protobuf/proto"

	"github.com/vanti-dev/sc-bos/internal/health/healthdb"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/historypb"
)

// These are the same as those in the historypb package.
const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

func parsePageInfo(req *gen.ListHealthCheckHistoryRequest) (nextID healthdb.RecordID, pageSize, totalSize int32, _ error) {
	pageToken, err := unmarshalToken(req.GetPageToken())
	if err != nil {
		return 0, 0, 0, err
	}
	if id := pageToken.GetRecordId(); id != "" {
		nextID, err = healthdb.ParseRecordID(id)
		if err != nil {
			return 0, 0, 0, err
		}
	}
	totalSize = pageToken.GetTotalSize()
	pageSize = req.GetPageSize()
	switch {
	case pageSize <= 0:
		pageSize = defaultPageSize
	case pageSize > maxPageSize:
		pageSize = maxPageSize
	}
	return nextID, pageSize, totalSize, nil
}

func unmarshalToken(token string) (*historypb.PageToken, error) {
	if token == "" {
		return &historypb.PageToken{}, nil
	}
	data, err := base64.RawStdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	pb := &historypb.PageToken{}
	err = proto.Unmarshal(data, pb)
	return pb, err
}

func createNextPageToken(r healthdb.Record, totalSize int32) (string, error) {
	return marshalToken(&historypb.PageToken{RecordId: r.ID.String(), TotalSize: totalSize})
}

func marshalToken(pb *historypb.PageToken) (string, error) {
	data, err := proto.Marshal(pb)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(data), nil
}
