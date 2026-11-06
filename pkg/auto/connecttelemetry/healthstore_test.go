package connecttelemetry

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timshannon/bolthold"
)

func TestBoltAckStore(t *testing.T) {
	db, err := bolthold.Open(filepath.Join(t.TempDir(), "db.bolt"), 0600, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	at := time.Date(2026, 11, 5, 18, 30, 0, 0, time.UTC)
	s := newBoltAckStore(db, healthStreamKey("tls://broker:8883", "node-1", "node-1", "tlm"))

	got, err := s.load()
	require.NoError(t, err)
	assert.Empty(t, got, "nothing saved yet")

	require.NoError(t, s.save([]healthAck{
		{Name: "dev1", Hash: "aaaa", AckedAt: at},
		{Name: "dev2", Hash: "bbbb", AckedAt: at.Add(time.Second)},
	}, nil))
	got, err = s.load()
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "aaaa", got["dev1"].Hash)
	assert.True(t, at.Equal(got["dev1"].AckedAt))
	assert.Equal(t, "dev2", got["dev2"].Name)

	// an update and a delete in one save, deleting what isn't there is fine
	require.NoError(t, s.save([]healthAck{{Name: "dev1", Hash: "cccc", AckedAt: at.Add(time.Minute)}}, []string{"dev2", "never"}))
	got, err = s.load()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "cccc", got["dev1"].Hash)
	assert.True(t, at.Add(time.Minute).Equal(got["dev1"].AckedAt))

	// another stream, say after a re-enrolment, starts afresh, and leaves the first alone
	other := newBoltAckStore(db, healthStreamKey("tls://broker:8883", "node-2", "node-2", "tlm"))
	got, err = other.load()
	require.NoError(t, err)
	assert.Empty(t, got)
	require.NoError(t, other.save([]healthAck{{Name: "dev1", Hash: "dddd", AckedAt: at}}, nil))
	got, err = s.load()
	require.NoError(t, err)
	assert.Equal(t, "cccc", got["dev1"].Hash)
}

func TestHealthStreamKey(t *testing.T) {
	base := healthStreamKey("tls://broker.example:8883", "exporter", "node-1", "tlm")
	assert.Equal(t, base, healthStreamKey("mqtts://Broker.example:8883", "exporter", "node-1", "tlm/"), "same broker and prefix")
	assert.NotEqual(t, base, healthStreamKey("tls://other.example:8883", "exporter", "node-1", "tlm"), "broker")
	assert.NotEqual(t, base, healthStreamKey("tls://broker.example:8883", "other", "node-1", "tlm"), "client id")
	// a configured client id stays the same across a re-enrolment, the node id doesn't
	assert.NotEqual(t, base, healthStreamKey("tls://broker.example:8883", "exporter", "node-2", "tlm"), "node id")
	assert.NotEqual(t, base, healthStreamKey("tls://broker.example:8883", "exporter", "node-1", "dev"), "topic prefix")
}
