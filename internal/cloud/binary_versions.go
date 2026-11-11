package cloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/google/renameio/v2/maybe"
)

// maxBinaryVersions bounds how many versions binaryVersions remembers. A rollback only ever returns to
// a recent version, so a few entries are enough.
const maxBinaryVersions = 10

// binaryVersions remembers the cloud's version id for each software version BOS was asked to run, so
// it can report the id once that version is running. Version strings identify builds. It is kept in a
// JSON file so it survives restarts.
type binaryVersions struct {
	path string
}

type binaryVersionEntry struct {
	Version   string `json:"version"`
	VersionID string `json:"versionId"`
}

// lookup returns the cloud's version id for version, or ok=false when there is none.
func (b *binaryVersions) lookup(version string) (id string, ok bool, err error) {
	entries, err := b.load()
	if err != nil {
		return "", false, err
	}
	i := slices.IndexFunc(entries, func(e binaryVersionEntry) bool { return e.Version == version })
	if i < 0 {
		return "", false, nil
	}
	return entries[i].VersionID, true, nil
}

// add records id as the cloud's version id for version, replacing any earlier id for it and
// forgetting the oldest versions beyond maxBinaryVersions.
func (b *binaryVersions) add(version, id string) error {
	if version == "" || id == "" {
		return nil
	}
	entries, err := b.load()
	if err != nil {
		// An unreadable record is replaced: keeping it would stop BOS recording anything again.
		entries = nil
	}
	entry := binaryVersionEntry{Version: version, VersionID: id}
	if len(entries) > 0 && entries[0] == entry {
		return nil
	}
	entries = slices.DeleteFunc(entries, func(e binaryVersionEntry) bool { return e.Version == version })
	entries = slices.Insert(entries, 0, entry)
	entries = entries[:min(len(entries), maxBinaryVersions)]

	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	return maybe.WriteFile(b.path, data, 0600)
}

// load returns the remembered versions, newest first.
func (b *binaryVersions) load() ([]binaryVersionEntry, error) {
	data, err := os.ReadFile(b.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var entries []binaryVersionEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode %s: %w", b.path, err)
	}
	return entries, nil
}
