package occupancyemail

import (
	"time"

	"github.com/smart-core-os/sc-bos/pkg/auto/occupancyemail/config"
)

type Attrs struct {
	Now   time.Time
	Stats []Stats
}

type Stats struct {
	Source    config.Source
	Last7Days OccupancyStats
	Days      []DayStats
}

type DayStats struct {
	Date time.Time
	OccupancyStats
}

type OccupancyStats struct {
	MaxPeopleCount int32
}
