package xovis

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/task"
)

type PushData struct {
	LogicsData *LogicsPushData `json:"logics_data,omitempty"`
}

type LogicsPushData struct {
	PackageInfo *PackageInfo `json:"package_info,omitempty"`
	SensorInfo  *SensorInfo  `json:"sensor_info,omitempty"`
	Logics      []Logic      `json:"logics"`
}

type Logic struct {
	ID         int           `json:"id"`
	Name       string        `json:"name"`
	Info       string        `json:"info"`
	Geometries []Geometry    `json:"geometries"`
	Records    []LogicRecord `json:"records"`
}

type LogicRecord struct {
	From            time.Time `json:"from"`
	To              time.Time `json:"to"`
	Samples         int       `json:"samples"`
	SamplesExpected int       `json:"samples_expected"`
	Counts          []Count   `json:"counts"`
}

type PackageInfo struct {
	Version   string    `json:"version"`
	ID        int       `json:"id"`
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	AgentID   int       `json:"agent_id"`
	AgentName string    `json:"agent_name"`
	AgentType string    `json:"agent_type"`
}

type SensorInfo struct {
	Type         string    `json:"type"`
	SerialNumber string    `json:"serial_number"`
	Name         string    `json:"name"`
	Group        string    `json:"group"`
	DeviceType   string    `json:"device_type"`
	HWVersion    string    `json:"hw_version"`
	SWVersion    string    `json:"sw_version"`
	Time         time.Time `json:"time"`
	Timezone     string    `json:"timezone"`
}

// readOnPush returns a task that, while attached, calls read whenever the sensor pushes data for logicID.
//
// Push records carry what happened during each closed resolution period (enter/leave deltas, not totals), while
// polls and reads carry the sensor's live totals. Adding a period's deltas to a total read during that period
// counts some people twice, so a push is used only as the cue to read: the value it leads to is always the
// sensor's own, and can never disagree with the next poll.
//
// Pushes that arrive while a read is in flight are folded into one more read, so a slow sensor never holds up
// the webhook.
func readOnPush(bus *minibus.Bus[PushData], logicID int, read func(context.Context) error) *task.Intermittent {
	return task.NewIntermittent(func(context.Context) (task.StopFn, error) {
		// outlives the Attach call that starts it, see task.StartFn
		ctx, stop := context.WithCancel(context.Background())
		pushes := bus.Listen(ctx)
		due := make(chan struct{}, 1)
		go func() {
			for data := range pushes {
				if data.LogicsData == nil {
					continue
				}
				if _, ok := findLogicRecords(data.LogicsData, logicID); !ok {
					continue
				}
				select {
				case due <- struct{}{}:
				default: // a read is already due
				}
			}
		}()
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-due:
					// a failed read is recorded against the device's health check, and the next poll retries
					_ = read(ctx)
				}
			}
		}()
		return stop, nil
	})
}
