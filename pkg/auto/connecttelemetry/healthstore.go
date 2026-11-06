package connecttelemetry

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/timshannon/bolthold"
	bolt "go.etcd.io/bbolt"
)

// healthAck is the last set the broker acknowledged for a resource, as saved across restarts.
type healthAck struct {
	Name    string
	Hash    string
	AckedAt time.Time
}

// healthAckStore saves which sets the broker has acknowledged, so a restart doesn't resend them.
type healthAckStore interface {
	// load returns the saved acknowledgements by resource name.
	load() (map[string]healthAck, error)
	// save writes put and removes del, all or nothing.
	save(put []healthAck, del []string) error
}

// nopAckStore saves nothing, so every set is sent again after a restart.
type nopAckStore struct{}

func (nopAckStore) load() (map[string]healthAck, error) { return nil, nil }
func (nopAckStore) save([]healthAck, []string) error    { return nil }

// healthAckBucket holds a nested bucket per stream, see healthStreamKey.
const healthAckBucket = "connecttelemetry-health-acks"

// boltAckStore keeps one stream's acknowledgements in the node's bolt database.
type boltAckStore struct {
	db     *bolthold.Store
	stream string
}

func newBoltAckStore(db *bolthold.Store, stream string) *boltAckStore {
	return &boltAckStore{db: db, stream: stream}
}

func (s *boltAckStore) load() (map[string]healthAck, error) {
	var recs []healthAck
	err := s.db.Bolt().View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(healthAckBucket))
		if b == nil {
			return nil
		}
		sb := b.Bucket([]byte(s.stream))
		if sb == nil {
			return nil
		}
		return s.db.FindInBucket(sb, &recs, nil)
	})
	if err != nil {
		return nil, err
	}
	res := make(map[string]healthAck, len(recs))
	for _, r := range recs {
		res[r.Name] = r
	}
	return res, nil
}

func (s *boltAckStore) save(put []healthAck, del []string) error {
	if len(put) == 0 && len(del) == 0 {
		return nil
	}
	return s.db.Bolt().Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(healthAckBucket))
		if err != nil {
			return err
		}
		sb, err := b.CreateBucketIfNotExists([]byte(s.stream))
		if err != nil {
			return err
		}
		for _, a := range put {
			if err := s.db.UpsertBucket(sb, a.Name, a); err != nil {
				return err
			}
		}
		for _, name := range del {
			if err := s.db.DeleteFromBucket(sb, name, healthAck{}); err != nil && !errors.Is(err, bolthold.ErrNotFound) {
				return err
			}
		}
		return nil
	})
}

// healthStreamKey identifies the stream of sets Connect receives from a node: what one broker
// acknowledged from one client, as one node id, under one topic prefix. A different broker, or a
// re-enrolment under another node id, starts afresh. The node id is part of the key as well as
// the client id because a configured client id stays the same across a re-enrolment, while Connect
// files the sets under the new node and holds none of them.
func healthStreamKey(host, clientID, nodeID, topicPrefix string) string {
	if u, err := url.Parse(host); err == nil && u.Host != "" {
		host = u.Host // so tls:// and mqtts:// to the same broker are one stream
	}
	return strings.Join([]string{strings.ToLower(host), clientID, nodeID, strings.TrimRight(topicPrefix, "/")}, "|")
}
