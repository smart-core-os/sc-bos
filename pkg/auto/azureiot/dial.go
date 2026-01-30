package azureiot

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/smart-core-os/sc-bos/internal/iothub"
	"github.com/smart-core-os/sc-bos/internal/iothub/auth"
	"github.com/smart-core-os/sc-bos/internal/iothub/dps"
)

type dialler interface {
	Dial(ctx context.Context) (iothub.Conn, error)
}

func diallerFromConfig(devCfg DeviceConfig, idScope string, grpKey auth.SASKey) (dialler, error) {
	if devCfg.UsesConnectionString() {
		connectionString := devCfg.ConnectionString
		if devCfg.ConnectionStringFile != "" {
			contents, err := os.ReadFile(devCfg.ConnectionStringFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read connection string file for device %q: %w", devCfg.Name, err)
			}
			connectionString = strings.TrimSpace(string(contents))
		}

		// the device specifies its own connection string, no need to use the DPS
		params, err := iothub.ParseConnectionString(connectionString)
		if err != nil {
			return nil, fmt.Errorf("invalid connection string for device %q: %w", devCfg.Name, err)
		}

		return &directDialler{params: params}, nil
	}

	regId := devCfg.RegistrationID
	if regId == "" {
		return nil, fmt.Errorf("device %q is missing a registration ID", devCfg.Name)
	}

	return &dpsDialler{
		idScope: idScope,
		regID:   regId,
		key:     grpKey,
	}, nil
}

type directDialler struct {
	params iothub.ConnectionParameters
}

func (d *directDialler) Dial(ctx context.Context) (iothub.Conn, error) {
	return iothub.Dial(ctx, d.params)
}

type dpsDialler struct {
	host    string
	idScope string
	regID   string
	key     auth.SASKey

	reg *dps.Registration
}

func (d *dpsDialler) Dial(ctx context.Context) (iothub.Conn, error) {
	if d.reg == nil {
		reg, err := dps.Register(ctx, d.host, d.idScope, d.regID, d.key)
		if err != nil {
			return nil, err
		}
		d.reg = &reg
	}

	params := d.reg.AsConnectionParameters()
	return iothub.Dial(ctx, params)
}
