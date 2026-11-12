package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	t.Run("defaults with file-path certs", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"type": "connecttelemetry",
			"traits": ["smartcore.bos.Meter"],
			"mqtt": {
				"host": "tls://broker:8883",
				"clientCertPath": "/c.crt",
				"clientKeyPath": "/c.key"
			}
		}`))
		require.NoError(t, err)
		assert.Equal(t, DefaultTopicPrefix, root.Mqtt.TopicPrefix)
		require.NotNil(t, root.Mqtt.Qos)
		assert.Equal(t, 1, *root.Mqtt.Qos)
		require.NotNil(t, root.Mqtt.MetadataInterval)
		assert.Equal(t, 100, *root.Mqtt.MetadataInterval)
		assert.NotNil(t, root.Mqtt.SendInterval)
		assert.Equal(t, 5.0, root.FetchTimeout.Seconds())
		assert.Equal(t, "dbo", root.PointNaming, "pointNaming defaults to dbo")
		assert.False(t, root.HealthEnabled(), "health defaults to off")
		assert.False(t, root.IncludeZones, "zones are skipped by default")
	})

	t.Run("includeZones", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Meter"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"includeZones": true
		}`))
		require.NoError(t, err)
		assert.True(t, root.IncludeZones)
	})

	t.Run("devices", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Meter", "smartcore.traits.OccupancySensor"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"devices": {"smartcore.traits.OccupancySensor": ["zones/treehouse"]}
		}`))
		require.NoError(t, err)
		assert.Equal(t, map[string][]string{"smartcore.traits.OccupancySensor": {"zones/treehouse"}}, root.Devices)
	})

	t.Run("devices for a trait not exported", func(t *testing.T) {
		_, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Meter"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"devices": {"smartcore.traits.OccupancySensor": ["zones/treehouse"]}
		}`))
		assert.ErrorContains(t, err, "not in traits")
	})

	t.Run("zones", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Meter", "smartcore.traits.OccupancySensor"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"zones": {"smartcore.traits.OccupancySensor": ["zones/treehouse"]}
		}`))
		require.NoError(t, err)
		assert.Equal(t, map[string][]string{"smartcore.traits.OccupancySensor": {"zones/treehouse"}}, root.Zones)
		assert.Empty(t, root.Devices)
	})

	for name, tc := range map[string]struct{ cfg, err string }{
		"zones for a trait not exported": {
			cfg: `"traits": ["smartcore.bos.Meter"], "zones": {"smartcore.traits.OccupancySensor": ["zones/treehouse"]}`,
			err: "zones has an entry for \"smartcore.traits.OccupancySensor\", which is not in traits",
		},
		"zones with an empty list": {
			cfg: `"traits": ["smartcore.traits.OccupancySensor"], "zones": {"smartcore.traits.OccupancySensor": []}`,
			err: "zones[\"smartcore.traits.OccupancySensor\"] is empty",
		},
		"zones and devices for one trait": {
			cfg: `"traits": ["smartcore.traits.OccupancySensor"],
				"devices": {"smartcore.traits.OccupancySensor": ["pir-01"]},
				"zones": {"smartcore.traits.OccupancySensor": ["zones/treehouse"]}`,
			err: "has both a devices and a zones entry",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseConfig([]byte(`{"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true}, ` + tc.cfg + `}`))
			assert.ErrorContains(t, err, tc.err)
		})
	}

	t.Run("zones and devices for different traits", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Meter", "smartcore.traits.OccupancySensor"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"devices": {"smartcore.bos.Meter": ["meter-01"]},
			"zones": {"smartcore.traits.OccupancySensor": ["zones/treehouse"]}
		}`))
		require.NoError(t, err)
		assert.Len(t, root.Devices, 1)
		assert.Len(t, root.Zones, 1)
	})

	t.Run("groupChildren", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Meter", "smartcore.bos.Temperature"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"groupChildren": ["smartcore.bos.Temperature"]
		}`))
		require.NoError(t, err)
		assert.Equal(t, []string{"smartcore.bos.Temperature"}, root.GroupChildren)
	})

	t.Run("groupChildren defaults to none", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Temperature"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true}
		}`))
		require.NoError(t, err)
		assert.Empty(t, root.GroupChildren)
	})

	t.Run("groupChildren for a trait not exported", func(t *testing.T) {
		_, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Meter"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"groupChildren": ["smartcore.bos.Temperature"]
		}`))
		assert.ErrorContains(t, err, "not in traits")
	})

	t.Run("groupChildren for a trait that can't be grouped", func(t *testing.T) {
		_, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Meter"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"groupChildren": ["smartcore.bos.Meter"]
		}`))
		assert.ErrorContains(t, err, "can't be grouped")
	})

	t.Run("sendOnChange heartbeat defaults to an hour", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Meter", "smartcore.traits.OccupancySensor", "smartcore.traits.AirTemperature"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"sendOnChange": {"smartcore.traits.OccupancySensor": {}, "smartcore.traits.AirTemperature": null}
		}`))
		require.NoError(t, err)
		require.Len(t, root.SendOnChange, 2)
		for name, oc := range root.SendOnChange {
			require.NotNil(t, oc, name)
			require.NotNil(t, oc.Heartbeat, name)
			assert.Equal(t, time.Hour, oc.Heartbeat.Duration, name)
		}
		assert.NotContains(t, root.SendOnChange, "smartcore.bos.Meter", "traits without an entry stay on the grid")
	})

	t.Run("sendOnChange keeps a heartbeat", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"traits": ["smartcore.traits.OccupancySensor"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"sendOnChange": {"smartcore.traits.OccupancySensor": {"heartbeat": "15m"}}
		}`))
		require.NoError(t, err)
		assert.Equal(t, 15*time.Minute, root.SendOnChange["smartcore.traits.OccupancySensor"].Heartbeat.Duration)
	})

	for name, tc := range map[string]struct{ cfg, err string }{
		"sendOnChange for a trait not exported": {
			cfg: `"traits": ["smartcore.bos.Meter"], "sendOnChange": {"smartcore.traits.OccupancySensor": {}}`,
			err: "not in traits",
		},
		"sendOnChange for meters": {
			cfg: `"traits": ["smartcore.bos.Meter"], "sendOnChange": {"smartcore.bos.Meter": {}}`,
			err: "meter's liveness",
		},
		"sendOnChange with a zero heartbeat": {
			cfg: `"traits": ["smartcore.traits.OccupancySensor"], "sendOnChange": {"smartcore.traits.OccupancySensor": {"heartbeat": "0s"}}`,
			err: "must be positive",
		},
		"sendOnChange with a negative heartbeat": {
			cfg: `"traits": ["smartcore.traits.OccupancySensor"], "sendOnChange": {"smartcore.traits.OccupancySensor": {"heartbeat": "-5m"}}`,
			err: "must be positive",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseConfig([]byte(`{"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true}, ` + tc.cfg + `}`))
			assert.ErrorContains(t, err, tc.err)
		})
	}

	t.Run("devices with an empty list", func(t *testing.T) {
		_, err := ParseConfig([]byte(`{
			"traits": ["smartcore.traits.OccupancySensor"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"devices": {"smartcore.traits.OccupancySensor": []}
		}`))
		assert.ErrorContains(t, err, "is empty")
	})

	t.Run("health defaults", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Meter"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"health": {"enabled": true}
		}`))
		require.NoError(t, err)
		require.True(t, root.HealthEnabled())
		assert.Equal(t, 15*time.Minute, root.Health.ManifestInterval.Duration)
		assert.Equal(t, 10.0, root.Health.MaxPublishRate)
		assert.False(t, root.Health.IncludeZones, "zone health is off by default")
	})

	t.Run("health includes zones", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"health": {"enabled": true, "includeZones": true}
		}`))
		require.NoError(t, err)
		assert.True(t, root.Health.IncludeZones)
		assert.False(t, root.IncludeZones, "independent of zone telemetry")
	})

	t.Run("health only", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"health": {"enabled": true, "manifestInterval": "1m", "maxPublishRate": 2.5}
		}`))
		require.NoError(t, err)
		assert.Empty(t, root.Traits)
		assert.Equal(t, time.Minute, root.Health.ManifestInterval.Duration)
		assert.Equal(t, 2.5, root.Health.MaxPublishRate)
	})

	t.Run("neither traits nor health", func(t *testing.T) {
		for _, cfg := range []string{
			`{"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true}}`,
			`{"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true}, "health": {"enabled": false}}`,
		} {
			_, err := ParseConfig([]byte(cfg))
			require.Error(t, err, cfg)
			assert.Contains(t, err.Error(), "nothing to publish")
		}
	})

	t.Run("negative health rate", func(t *testing.T) {
		_, err := ParseConfig([]byte(`{
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
			"health": {"enabled": true, "maxPublishRate": -1}
		}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxPublishRate")
	})

	t.Run("health bounds", func(t *testing.T) {
		for _, tc := range []struct {
			health  string
			wantErr string
		}{
			{health: `"manifestInterval": "15m", "maxPublishRate": 16`},
			{health: `"manifestInterval": "16m"`, wantErr: "manifestInterval"},
			{health: `"maxPublishRate": 16.5`, wantErr: "maxPublishRate"},
		} {
			_, err := ParseConfig([]byte(`{
				"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true},
				"health": {"enabled": true, ` + tc.health + `}
			}`))
			if tc.wantErr == "" {
				assert.NoError(t, err, tc.health)
			} else if assert.Error(t, err, tc.health) {
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		}
	})

	t.Run("pointNaming raw is accepted", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Meter"],
			"pointNaming": "raw",
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true}
		}`))
		require.NoError(t, err)
		assert.Equal(t, "raw", root.PointNaming)
	})

	t.Run("invalid pointNaming rejected", func(t *testing.T) {
		_, err := ParseConfig([]byte(`{
			"pointNaming": "vendor",
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true}
		}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pointNaming")
	})

	t.Run("cloud credential mode", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
			"traits": ["smartcore.bos.Meter"],
			"mqtt": {"host": "tls://broker:8883", "useCloudCredential": true, "topicPrefix": "tlm/site-a"}
		}`))
		require.NoError(t, err)
		assert.True(t, root.Mqtt.UseCloudCredential)
		assert.Equal(t, "tlm/site-a", root.Mqtt.TopicPrefix)
	})

	t.Run("host is required", func(t *testing.T) {
		_, err := ParseConfig([]byte(`{"mqtt": {"useCloudCredential": true}}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "host")
	})

	t.Run("credential modes are mutually exclusive", func(t *testing.T) {
		_, err := ParseConfig([]byte(`{
			"mqtt": {"host": "tls://b:8883", "useCloudCredential": true, "clientCertPath": "/c.crt"}
		}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "useCloudCredential cannot be combined")
	})

	t.Run("file-path mode needs cert and key", func(t *testing.T) {
		_, err := ParseConfig([]byte(`{
			"mqtt": {"host": "tls://b:8883", "clientCertPath": "/c.crt"}
		}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "clientCertPath and mqtt.clientKeyPath")
	})

	t.Run("qos out of range", func(t *testing.T) {
		_, err := ParseConfig([]byte(`{
			"mqtt": {"host": "tls://b:8883", "useCloudCredential": true, "qos": 3}
		}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "qos")
	})
}
