package xovis

import (
	"time"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

type countAccumulator struct {
	forwardCountID     int
	backwardCountID    int
	forwardCountValue  int
	backwardCountValue int
}

type countEvent struct {
	time time.Time
	// We associate the Xovis API's "forward" events with "enter" events in Smart Core,
	// and Xovis "backward" events with "leave" events in Smart Core.
	direction traits.EnterLeaveEvent_Direction
}

func (c *countAccumulator) consumeRecords(records ...LogicRecord) ([]countEvent, error) {
	var events []countEvent
	for _, record := range records {
		fwDelta, ok := findCountValueByID(record.Counts, c.forwardCountID)
		if ok {
			c.forwardCountValue += fwDelta
			event := countEvent{
				time:      record.To,
				direction: traits.EnterLeaveEvent_ENTER,
			}
			for range fwDelta {
				events = append(events, event)
			}
		}

		bwDelta, ok := findCountValueByID(record.Counts, c.backwardCountID)
		if ok {
			c.backwardCountValue += bwDelta // always write this, if the count is reset we only want to desync once
			event := countEvent{
				time:      record.To,
				direction: traits.EnterLeaveEvent_LEAVE,
			}
			for range bwDelta {
				events = append(events, event)
			}
		}
	}
	return events, nil
}

func findCountByName(counts []Count, name string) (id, value int, ok bool) {
	for _, count := range counts {
		if count.Name == name {
			return count.ID, count.Value, true
		}
	}
	return
}

func findCountValueByID(counts []Count, id int) (value int, ok bool) {
	for _, count := range counts {
		if count.ID == id {
			return count.Value, true
		}
	}
	return
}
