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
