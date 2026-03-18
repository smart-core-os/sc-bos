// Deprecated: use the vendingpb package directly.
package unitpb

import "github.com/smart-core-os/sc-bos/pkg/proto/vendingpb"

// Deprecated: use vendingpb.ConvertUnit.
func Convert(v float64, from, to vendingpb.Consumable_Unit) (float64, error) {
	return vendingpb.ConvertUnit(v, from, to)
}

// Deprecated: use vendingpb.ConvertUnit32.
func Convert32(v float32, from, to vendingpb.Consumable_Unit) (float32, error) {
	return vendingpb.ConvertUnit32(v, from, to)
}
