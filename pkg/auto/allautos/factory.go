package allautos

import (
	"github.com/vanti-dev/sc-bos/pkg/auto"
	"github.com/vanti-dev/sc-bos/pkg/auto/azureiot"
	"github.com/vanti-dev/sc-bos/pkg/auto/bms"
	"github.com/vanti-dev/sc-bos/pkg/auto/export"
	"github.com/vanti-dev/sc-bos/pkg/auto/exporthttp"
	"github.com/vanti-dev/sc-bos/pkg/auto/healthbounds"
	"github.com/vanti-dev/sc-bos/pkg/auto/history"
	"github.com/vanti-dev/sc-bos/pkg/auto/lights"
	"github.com/vanti-dev/sc-bos/pkg/auto/meteremail"
	"github.com/vanti-dev/sc-bos/pkg/auto/notificationsemail"
	"github.com/vanti-dev/sc-bos/pkg/auto/occupancyemail"
	"github.com/vanti-dev/sc-bos/pkg/auto/resetbrightness"
	"github.com/vanti-dev/sc-bos/pkg/auto/resetenterleave"
	"github.com/vanti-dev/sc-bos/pkg/auto/statusalerts"
	"github.com/vanti-dev/sc-bos/pkg/auto/statusemail"
	"github.com/vanti-dev/sc-bos/pkg/auto/udmi"
)

// Factories returns a new map containing all known auto factories.
func Factories() map[string]auto.Factory {
	return map[string]auto.Factory{
		azureiot.FactoryName:        azureiot.Factory,
		bms.AutoType:                bms.Factory,
		"export-mqtt":               export.MQTTFactory,
		healthbounds.AutoName:       healthbounds.Factory,
		"history":                   history.Factory,
		lights.AutoType:             lights.Factory,
		meteremail.AutoName:         meteremail.Factory,
		notificationsemail.AutoName: notificationsemail.Factory,
		occupancyemail.AutoName:     occupancyemail.Factory,
		resetbrightness.AutoName:    resetbrightness.Factory,
		resetenterleave.AutoName:    resetenterleave.Factory,
		statusalerts.AutoName:       statusalerts.Factory,
		statusemail.AutoName:        statusemail.Factory,
		udmi.AutoType:               udmi.Factory,
		exporthttp.AutoName:         exporthttp.Factory,
	}
}
