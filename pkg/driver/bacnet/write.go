package bacnet

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"go.uber.org/multierr"

	"github.com/smart-core-os/sc-bos/pkg/app/appconf"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	historyconfig "github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

// ConfigForFloor is basically just a wrapper around the root config with support for automations and traits
type ConfigForFloor struct {
	config.Root
	Devices      []config.Device
	Automations  []map[string]any // map looks like config that embeds auto.Config
	Traits       []map[string]any // map looks like config that embeds config.Trait
	AddHistories []trait.Name
}

func (bc *ConfigForFloor) addHistoryForTraits(traits []map[string]any, pollingSchedule *jsontypes.Schedule) []map[string]any {
	var allConfigs []map[string]any
	for _, t := range traits {
		if kind, ok := t["kind"]; ok {
			for _, traitWithHistory := range bc.AddHistories {
				if kind == traitWithHistory {
					name := t["name"].(string)
					c := make(map[string]any)
					c["name"] = strings.Replace(name, "devices", "history", 1)
					c["type"] = "history"
					c["source"] = &historyconfig.Source{
						Name:            name,
						Trait:           traitWithHistory,
						PollingSchedule: pollingSchedule,
					}
					c["storage"] = &historyconfig.Storage{
						Type: "hub",
					}
					allConfigs = append(allConfigs, c)
				}
			}
		}
	}
	return allConfigs
}

// WriteBacnetConfig writes bacnet config for each floor to the given directories for each floor
// configPerFloor is a map of floor name to the bacnet config for that floor
// dirForFloor is a map of floor name to the directory for that floor. They keys are the same as configPerFloor and
// there must be a directory value defined for each floor. This allows you to put 2 floor configs in the same directory.
// configRoot is the root directory for the config, defaults to "config" if empty
// scPrefix is the prefix for the driver's Name. The bacnet driver name will become <sc-prefix>/floor-xx/drivers/<subsystem>
// subsystem is the name of the subsystem, e.g. "bms", "fire-alarm", etc. which gets used as the file prefix for the output files
// pollingSchedule is a cron schedule for polling the history traits. If nil, no polling will be added and the history auto will default to pulling records on change.
func WriteBacnetConfig(configPerFloor map[string]*ConfigForFloor, dirForFloor map[string]string, configRoot string,
	scPrefix string, subsystem string, pollingSchedule *jsontypes.Schedule) error {
	if configRoot == "" {
		configRoot = "config"
	}
	for floor, cfg := range configPerFloor {

		if len(cfg.AddHistories) > 0 {
			autos := cfg.addHistoryForTraits(cfg.Traits, pollingSchedule)
			cfg.Automations = append(cfg.Automations, autos...)
		}

		sortDevices(cfg.Devices)
		sortRawConfig(cfg.Traits)
		sortRawConfig(cfg.Automations)

		configDir, ok := dirForFloor[floor]
		if !ok {
			return fmt.Errorf("no config dir for floor %s", floor)
		}
		configDir = path.Join(configRoot, configDir)

		err := writeToDir(configDir, floor, scPrefix, cfg, subsystem)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeToDir(dir string, floor string, scPrefix string, bacnetCfg *ConfigForFloor, subsystem string) error {
	var errs error
	outFile := filepath.Join(dir, fmt.Sprintf("%s.%s.part.json", floor, subsystem))
	cfg := appconf.Config{}
	driverCfg := config.Defaults()
	// copy over any customisations we have made
	driverCfg = bacnetCfg.Root
	driverCfg.BaseConfig = driver.BaseConfig{
		Name: path.Join(scPrefix, fmt.Sprintf("floor-%s", floor), "drivers", subsystem),
		Type: DriverName,
	}

	driverCfg.SystemHealth = config.Health{
		OccupantImpact:  config.OccupantImpact(healthpb.HealthCheck_NO_OCCUPANT_IMPACT),
		EquipmentImpact: config.EquipmentImpact(healthpb.HealthCheck_NO_EQUIPMENT_IMPACT),
	}

	driverCfg.Devices = bacnetCfg.Devices

	for _, t := range bacnetCfg.Traits {
		rawTrait, err := marshalTrait(t)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		driverCfg.Traits = append(driverCfg.Traits, rawTrait)
	}

	rawConfig, err := marshalDriver(driverCfg)
	if errs != nil {
		errs = multierr.Append(errs, err)
	} else {
		cfg.Drivers = append(cfg.Drivers, rawConfig)
	}

	for _, automation := range bacnetCfg.Automations {
		rawConfig, err := marshalAuto(automation)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		cfg.Automation = append(cfg.Automation, rawConfig)
	}

	if errs != nil {
		return errs
	}

	file, err := os.OpenFile(outFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

func marshalTrait(data map[string]any) (config.RawTrait, error) {
	t := config.Trait{
		Name: data["name"].(string),
		Kind: data["kind"].(trait.Name),
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return config.RawTrait{}, err
	}
	return config.RawTrait{
		Trait: t,
		Raw:   raw,
	}, nil
}

func marshalAuto(data map[string]any) (auto.RawConfig, error) {
	c := auto.Config{
		Name: data["name"].(string),
		Type: data["type"].(string),
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return auto.RawConfig{}, err
	}
	return auto.RawConfig{
		Config: c,
		Raw:    raw,
	}, nil
}

func marshalDriver(data config.Root) (driver.RawConfig, error) {
	d := data.BaseConfig
	raw, err := json.Marshal(data)
	if err != nil {
		return driver.RawConfig{}, err
	}
	return driver.RawConfig{
		BaseConfig: d,
		Raw:        raw,
	}, nil
}

func sortDevices(devices []config.Device) {
	for _, device := range devices {
		sort.Slice(device.Objects, func(i, j int) bool {
			ida := device.Objects[i].ID
			idb := device.Objects[j].ID
			if ida.Type < idb.Type {
				return true
			}
			if ida.Type > idb.Type {
				return false
			}
			return ida.Instance < idb.Instance
		})
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Name < devices[j].Name
	})
}

func sortRawConfig(rawConfig []map[string]any) {
	sort.Slice(rawConfig, func(i, j int) bool {
		n1, n2 := rawConfig[i]["name"].(string), rawConfig[j]["name"].(string)
		if n1 == n2 {
			if _, ok := rawConfig[i]["kind"]; ok {
				if _, ok2 := rawConfig[j]["kind"]; ok2 {
					return rawConfig[i]["kind"].(trait.Name) < rawConfig[j]["kind"].(trait.Name)
				}
			} else if _, ok := rawConfig[i]["type"]; ok {
				if _, ok2 := rawConfig[j]["type"]; ok2 {
					return rawConfig[i]["type"].(trait.Name) < rawConfig[j]["type"].(trait.Name)
				}
			}
		}
		return n1 < n2
	})
}
