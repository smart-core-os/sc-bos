package meteremail

import (
	"sort"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/auto/meteremail/config"
)

const MeterTypeWater = 1
const MeterTypeElectric = 2

type MeterType uint8

type Attrs struct {
	Now                  time.Time
	Stats                []Stats
	ReadingsByFloorZone  map[string]map[string][]Stats // map floor -> zone -> reading
	EnergySummaryReports []SummaryReport
	WaterSummaryReports  []SummaryReport
	TemplateArgs         config.TemplateArgs
}

// grab the floors and sort them so we can iterate over consistently
func (a *Attrs) getFloorKeys() []string {

	floorKeys := make([]string, 0, len(a.ReadingsByFloorZone))
	for k := range a.ReadingsByFloorZone {
		floorKeys = append(floorKeys, k)
	}
	sort.Strings(floorKeys)
	return floorKeys
}

type Stats struct {
	Source       config.Source
	MeterReading MeterReading
}

type MeterReading struct {
	MeterType MeterType
	Date      time.Time
	Reading   float32
}

type SummaryReport struct {
	Floor        string
	Zone         string
	TotalReading float32
}
