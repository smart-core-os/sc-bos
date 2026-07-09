package config

import (
	"testing"

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
	})

	t.Run("pointNaming raw is accepted", func(t *testing.T) {
		root, err := ParseConfig([]byte(`{
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
