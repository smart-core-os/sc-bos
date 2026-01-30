package allsystems

import (
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/alerts"
	"github.com/smart-core-os/sc-bos/pkg/system/authn"
	"github.com/smart-core-os/sc-bos/pkg/system/gateway"
	"github.com/smart-core-os/sc-bos/pkg/system/history"
	"github.com/smart-core-os/sc-bos/pkg/system/hub"
	"github.com/smart-core-os/sc-bos/pkg/system/publications"
	"github.com/smart-core-os/sc-bos/pkg/system/tenants"
)

// Factories returns a new map containing all known system factories.
func Factories() map[string]system.Factory {
	gatewayFactory := gateway.Factory()
	return map[string]system.Factory{
		"alerts":           alerts.Factory,
		"authn":            authn.Factory(),
		"history":          history.Factory,
		"hub":              hub.Factory(),
		gateway.Name:       gatewayFactory,
		gateway.LegacyName: gatewayFactory,
		"publications":     publications.Factory,
		"tenants":          tenants.Factory,
	}
}
