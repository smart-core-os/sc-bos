package merge

import (
	"testing"
)

func TestReadAirQualitySensorConfig_VOCDensity(t *testing.T) {
	raw := []byte(`{
		"name": "test",
		"voc": {"device": "d1", "object": "AnalogValue:1"},
		"vocDensity": {"device": "d1", "object": "AnalogValue:2", "scale": 1000}
	}`)
	cfg, err := readAirQualitySensorConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.VOC == nil {
		t.Fatal("voc not read")
	}
	if cfg.VOCDensity == nil {
		t.Fatal("vocDensity not read")
	}
	if got := cfg.VOCDensity.Scale; got != 1000 {
		t.Errorf("vocDensity scale = %v, want 1000", got)
	}
	// mg/m³ from the device must come out as µg/m³ on the trait.
	if got := cfg.VOCDensity.Scaled(float32(2.65)); got != float32(2650) {
		t.Errorf("vocDensity scaled 2.65 = %v, want 2650", got)
	}
	if got := cfg.VOC.Scale; got != 0 {
		t.Errorf("voc scale = %v, want unset", got)
	}
}
