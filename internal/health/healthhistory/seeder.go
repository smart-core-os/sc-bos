package healthhistory

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/internal/health/healthdb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// Seeder initialises health checks from historical data.
type Seeder struct {
	db SeederStore
}

// A SeederStore provides access to the last known health check history record.
type SeederStore interface {
	ReadLastRecord(ctx context.Context, id healthdb.CheckID) (healthdb.Record, error)
}

func NewSeeder(db SeederStore) *Seeder {
	return &Seeder{db: db}
}

func (s *Seeder) Seed(ctx context.Context, name string, c *gen_healthpb.HealthCheck) *gen_healthpb.HealthCheck {
	old, err := s.lastCheck(ctx, name, c.Id)
	if err != nil {
		return nil // no change made
	}
	healthpb.MergeCheck(proto.Merge, old, c)
	return old
}

func (s *Seeder) lastCheck(ctx context.Context, name, id string) (*gen_healthpb.HealthCheck, error) {
	oldDBRecord, err := s.db.ReadLastRecord(ctx, healthdb.CheckID{Name: name, ID: id})
	if err != nil {
		return nil, err
	}
	oldHistRecord, err := decodeRecord(oldDBRecord)
	if err != nil {
		return nil, err
	}
	return oldHistRecord.GetHealthCheck(), nil
}
