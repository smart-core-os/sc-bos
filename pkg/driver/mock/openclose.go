package mock

import (
	"encoding/json"

	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/mock/auto"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait/openclosepb"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

// mockOpenClose returns a mock OpenClose device and automation.
//
// Configuration of the mock device is done via the trait metadata more map.
// You can specify a "presets" key which has the JSON format of `[{name, title?, positions}, ...]`,
// where `positions` is the protojson representation of either a single or array of traits.OpenClosePosition.
func mockOpenClose(traitMd *traits.TraitMetadata, deviceName string, logger *zap.Logger) ([]wrap.ServiceUnwrapper, service.Lifecycle) {
	var opts []resource.Option

	opts = append(opts, parseOpenClosePresets(traitMd, deviceName, logger)...)

	model := openclosepb.NewModel(opts...)
	server := openclosepb.NewModelServer(model)
	return []wrap.ServiceUnwrapper{openclosepb.WrapApi(server), openclosepb.WrapInfo(server)}, auto.OpenClose(model)
}

func parseOpenClosePresets(traitMd *traits.TraitMetadata, deviceName string, logger *zap.Logger) []resource.Option {
	presets, ok := traitMd.GetMore()["presets"]
	if !ok {
		return nil
	}
	type presetJson struct {
		Name      string          `json:"name,omitempty"`
		Title     string          `json:"title,omitempty"`
		Positions json.RawMessage `json:"positions,omitempty"`
	}
	var cfg []presetJson
	if err := json.Unmarshal([]byte(presets), &cfg); err != nil {
		logger.Sugar().Warnf("Unable to unmarshal presets for mock device %q: %v. %v", deviceName, err, presets)
		return nil
	}

	var opts []resource.Option
	for _, presetCfg := range cfg {
		if presetCfg.Name == "" {
			logger.Sugar().Warnf("No name provided for mock device preset for %q", deviceName)
			continue
		}
		if len(presetCfg.Positions) == 0 {
			logger.Sugar().Warnf("No positions provided for mock device preset %s for %q", presetCfg.Name, deviceName)
			continue
		}

		// support both "positions": [{...}] and "positions": {...}
		var positionsJson []json.RawMessage
		switch presetCfg.Positions[0] {
		case '[':
			if err := json.Unmarshal(presetCfg.Positions, &positionsJson); err != nil {
				logger.Sugar().Warnf("Unable to unmarshal positions for mock device %q: %v. %v", deviceName, err, presetCfg.Positions)
				continue
			}
			if len(positionsJson) == 0 {
				logger.Sugar().Warnf("No positions provided for mock device preset %s for %q", presetCfg.Name, deviceName)
				continue
			}
		case '{':
			positionsJson = []json.RawMessage{presetCfg.Positions}
		default:
			logger.Sugar().Warnf("Invalid positions format for mock device preset %s for %q: %v", presetCfg.Name, deviceName, presetCfg.Positions)
			continue
		}

		positions := make([]*traits.OpenClosePosition, len(positionsJson))
		for i, posJson := range positionsJson {
			pos := &traits.OpenClosePosition{}
			if err := protojson.Unmarshal(posJson, pos); err != nil {
				logger.Sugar().Warnf("Unable to unmarshal position %s.%d for mock device %q: %v. %v", presetCfg.Name, i, deviceName, err, posJson)
				continue
			}
			positions[i] = pos
		}
		if len(positions) == 0 {
			continue // errors will already have been logged
		}

		desc := &traits.OpenClosePositions_Preset{
			Name:  presetCfg.Name,
			Title: presetCfg.Title,
		}
		opts = append(opts, openclosepb.WithPreset(desc, positions...))
	}
	return opts
}
