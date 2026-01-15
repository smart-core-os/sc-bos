package merge

import (
	"context"
	"errors"
	"fmt"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	gen_statuspb "github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
)

func ptr[T any](v T, err error) (*T, error) {
	return &v, err
}

func valuesEquivalent(a, b any) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func readPropertyFloat32(ctx context.Context, client *gobacnet.Client, known known.Context, value config.ValueSource) (float32, error) {
	data, err := comm.ReadProperty(ctx, client, known, value)
	if err != nil {
		return 0, err
	}
	return comm.Float32Value(data)
}

var (
	ErrTraitNotSupported = errors.New("trait not supported")
)

func initTraitStatus(statuses *statuspb.Map, name, trait string) {
	statuses.UpdateProblem(name, &gen_statuspb.StatusLog_Problem{
		Name:        name + ":" + trait,
		Level:       gen_statuspb.StatusLog_NOMINAL,
		Description: "Waiting for first interaction",
	})
}
