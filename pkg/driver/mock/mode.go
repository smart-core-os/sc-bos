package mock

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait/modepb"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

// mockMode returns a mock Mode device and automation.
//
// The mode device can be configured via the trait metadata more map.
// You can specify a "modes" key which has the JSON format of `{modes:[{name, ordered?, values}, ...]}`,
// this format is the protojson serialized form of traits.Modes.
func mockMode(traitMd *traits.TraitMetadata, deviceName string, logger *zap.Logger) ([]wrap.ServiceUnwrapper, service.Lifecycle) {
	var model *modepb.Model
	if modes, err := parseModes(traitMd); modes != nil {
		model = modepb.NewModelModes(modes)
	} else {
		if err != nil {
			logger.Warn("Unable to parse modes for mock device", zap.String("device", deviceName), zap.Error(err))
		}
		model = modepb.NewModel()
	}
	modes := model.Modes()
	infoServer := &modepb.InfoServer{Modes: &traits.ModesSupport{AvailableModes: modes}}
	return []wrap.ServiceUnwrapper{modepb.WrapApi(modepb.NewModelServer(model)), modepb.WrapInfo(infoServer)}, nil
}

func parseModes(traitMd *traits.TraitMetadata) (*traits.Modes, error) {
	modesJson, ok := traitMd.GetMore()["modes"]
	if !ok || modesJson == "" {
		return nil, nil
	}
	modes := &traits.Modes{}
	err := protojson.Unmarshal([]byte(modesJson), modes)
	if err != nil {
		return nil, err
	}
	return modes, nil
}
