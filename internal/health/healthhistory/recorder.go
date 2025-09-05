package healthhistory

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/vanti-dev/sc-bos/internal/health/healthdb"
	"github.com/vanti-dev/sc-bos/pkg/gen"
)

// A Recorder records health check updates into a database.
type Recorder struct {
	db RecorderStore
}

// A RecorderStore allows inserting health check history records.
type RecorderStore interface {
	Insert(ctx context.Context, record healthdb.Record) (healthdb.Record, error)
}

func NewRecorder(db RecorderStore) *Recorder {
	return &Recorder{db: db}
}

func (r *Recorder) Record(ctx context.Context, name string, check *gen.HealthCheck) error {
	main, aux, err := splitPayloads(check)
	if err != nil {
		return err
	}
	rec := healthdb.Record{
		Name:       name,
		CheckID:    check.Id,
		CreateTime: time.Now(),
		Main:       main,
		Aux:        aux,
	}
	_, err = r.db.Insert(ctx, rec)
	return err
}

func splitPayloads(c *gen.HealthCheck) (main, aux []byte, _ error) {
	m, a := splitCheck(c)
	main, err := proto.Marshal(m)
	if err != nil {
		return nil, nil, err
	}
	aux, err = proto.Marshal(a)
	if err != nil {
		return nil, nil, err
	}
	return main, aux, nil
}

// splitCheck divides the properties of c into main and aux messages.
// The main message contains frequently changing data, aux everything else.
// The passed c may be modified by this call.
func splitCheck(c *gen.HealthCheck) (main, aux *gen.HealthCheck) {
	// without benchmarks and real world stats, we assume most data is fairly static
	aux = c
	main = &gen.HealthCheck{}

	// copy frequently changing fields from aux to main
	if v := c.GetBounds().GetCurrentValue(); v != nil {
		main.Check = &gen.HealthCheck_Bounds_{Bounds: &gen.HealthCheck_Bounds{CurrentValue: v}}
		aux = proto.Clone(aux).(*gen.HealthCheck)
		aux.GetBounds().CurrentValue = nil
	}
	return main, aux
}

// RecordOnUpdate returns a function that records health check updates with a timeout.
// The returned function signature matches the OnUpdate callback for the [*healthpb.Registry].
func (r *Recorder) RecordOnUpdate(timeout time.Duration, onErr func(error)) func(string, *gen.HealthCheck) {
	return func(s string, check *gen.HealthCheck) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := r.Record(ctx, s, check)
		if err != nil && onErr != nil {
			onErr(err)
		}
	}
}
