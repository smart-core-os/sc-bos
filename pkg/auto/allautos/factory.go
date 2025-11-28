package allautos

import (
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/azureiot"
	"github.com/smart-core-os/sc-bos/pkg/auto/bms"
	"github.com/smart-core-os/sc-bos/pkg/auto/export"
	"github.com/smart-core-os/sc-bos/pkg/auto/exporthttp"
	"github.com/smart-core-os/sc-bos/pkg/auto/history"
	"github.com/smart-core-os/sc-bos/pkg/auto/lights"
	"github.com/smart-core-os/sc-bos/pkg/auto/meteremail"
	"github.com/smart-core-os/sc-bos/pkg/auto/notificationsemail"
	"github.com/smart-core-os/sc-bos/pkg/auto/occupancyemail"
	"github.com/smart-core-os/sc-bos/pkg/auto/resetbrightness"
	"github.com/smart-core-os/sc-bos/pkg/auto/resetenterleave"
	"github.com/smart-core-os/sc-bos/pkg/auto/sccexporter"
	"github.com/smart-core-os/sc-bos/pkg/auto/statusalerts"
	"github.com/smart-core-os/sc-bos/pkg/auto/statusemail"
	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
)

// Factories returns a new map containing all known auto factories.
func Factories() map[string]auto.Factory {
	return map[string]auto.Factory{
		azureiot.FactoryName:        azureiot.Factory,
		bms.AutoType:                bms.Factory,
		"export-mqtt":               export.MQTTFactory,
		"history":                   history.Factory,
		lights.AutoType:             lights.Factory,
		meteremail.AutoName:         meteremail.Factory,
		notificationsemail.AutoName: notificationsemail.Factory,
		occupancyemail.AutoName:     occupancyemail.Factory,
		resetbrightness.AutoName:    resetbrightness.Factory,
		resetenterleave.AutoName:    resetenterleave.Factory,
		sccexporter.AutoName:        sccexporter.Factory,
		statusalerts.AutoName:       statusalerts.Factory,
		statusemail.AutoName:        statusemail.Factory,
		udmi.AutoType:               udmi.Factory,
		exporthttp.AutoName:         exporthttp.Factory,
	}
}
