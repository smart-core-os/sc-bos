package udmi

import "testing"

func TestIsEventTopic(t *testing.T) {
	tests := map[string]bool{
		// pointset/event topics: ephemeral, not retained
		"client/site-01/HVAC/PICV-12345/event/pointset/points": true,
		"client/site-01/HVAC/PICV-12345/events/pointset":       true,
		// state and metadata: retained
		"client/site-01/HVAC/PICV-12345/state":         false,
		"client/site-01/HVAC/PICV-12345/metadata.json": false,
		"client/site-01/HVAC/PICV-12345/metadata":      false,
	}
	for topic, want := range tests {
		if got := isEventTopic(topic); got != want {
			t.Errorf("isEventTopic(%q) = %v, want %v", topic, got, want)
		}
	}
}
