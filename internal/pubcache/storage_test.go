package pubcache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/publicationpb"
)

func TestMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	t.Run("rejectsNilPublication", func(t *testing.T) {
		_ = rejectsNilPublication(ctx, t, storage)
	})
	t.Run("rejectsEmptyPublicationID", func(t *testing.T) {
		_ = rejectsEmptyPublicationID(ctx, t, storage)
	})
	t.Run("publicationRoundTrip", func(t *testing.T) {
		_ = publicationRoundTrip(ctx, t, storage)
	})
}

// checks the full lifecycle of a publication - insert, update, retrieve, delete
func publicationRoundTrip(ctx context.Context, t *testing.T, storage Storage) (ok bool) {
	id := "publicationRoundTrip!:$@.-_"
	baseTime := time.Date(2022, 7, 18, 11, 27, 0, 0, time.UTC)
	pub1 := &publicationpb.Publication{
		Id:          id,
		Version:     "1",
		Body:        []byte("1"),
		PublishTime: timestamppb.New(baseTime),
	}

	// first, check that there is no existing publication with the test ID
	_, err := storage.LoadPublication(ctx, id)
	if err == nil {
		t.Errorf("can't run test: publication %q already exists", id)
		return false
	} else if !errors.Is(err, ErrPublicationNotFound) {
		t.Errorf("LoadPublication(_, %q) error: %s", id, err.Error())
		return false
	}

	// insert the publication into the storage
	err = storage.StorePublication(ctx, pub1)
	if err != nil {
		t.Errorf("StorePublication(...) error: %s", err.Error())
		return false
	}

	// retrieve it, check it's the same
	actual, err := storage.LoadPublication(ctx, id)
	if err != nil {
		t.Errorf("LoadPublication(_, %q) error: %s", id, err)
		return false
	}
	if !proto.Equal(pub1, actual) {
		diff := cmp.Diff(pub1, actual)
		t.Errorf("Round-tripped publication is not equal (-want +got):\n%s", diff)
		return false
	}

	// update it with a different value
	pub2 := &publicationpb.Publication{
		Id:          id,
		Version:     "2",
		Body:        []byte("2"),
		PublishTime: timestamppb.New(baseTime.Add(time.Hour)),
	}
	err = storage.StorePublication(ctx, pub2)
	if err != nil {
		t.Errorf("StorePublication(...) error: %s", err.Error())
	}

	// retrieve it, check the update has applied
	actual, err = storage.LoadPublication(ctx, id)
	if err != nil {
		t.Errorf("LoadPublication(_, %q) error: %s", id, err)
		return false
	}
	if !proto.Equal(pub2, actual) {
		diff := cmp.Diff(pub2, actual)
		t.Errorf("Round-tripped publication is not equal (-want +got):\n%s", diff)
		return false
	}

	// delete the publication, it should be present
	present, err := storage.DeletePublication(ctx, id)
	if err != nil {
		t.Errorf("DeletePublication(_, %q) error: %s", id, err.Error())
		return false
	}
	if !present {
		t.Errorf("DeletePublication(_, %q) reported publication not present", id)
		return false
	}

	// delete it again, it's no longer present
	present, err = storage.DeletePublication(ctx, id)
	if err != nil {
		t.Errorf("DeletePublication(_, %q) error: %s", id, err.Error())
		return false
	}
	if present {
		t.Errorf("DeletePublication(_, %q) reported publication present", id)
		return false
	}

	return true
}

// checks that the storage returns ErrPublicationInvalid when storing a nil publication
func rejectsNilPublication(ctx context.Context, t *testing.T, storage Storage) (ok bool) {
	err := storage.StorePublication(ctx, nil)
	if errors.Is(err, ErrPublicationInvalid) {
		return true
	} else if err != nil {
		t.Errorf("for nil input, expected ErrPublicationInvalid but got error: %s", err.Error())
		return false
	} else {
		t.Error("for nil input, expected ErrPublicationInvalid but got nil error")
		return false
	}
}

// checks that the storage returns ErrPublicationInvalid when storing a publication with an empty ID string
func rejectsEmptyPublicationID(ctx context.Context, t *testing.T, storage Storage) (ok bool) {
	pub := &publicationpb.Publication{Id: ""}
	err := storage.StorePublication(ctx, pub)
	if errors.Is(err, ErrPublicationInvalid) {
		return true
	} else if err != nil {
		t.Errorf("for nil input, expected ErrPublicationInvalid but got error: %s", err.Error())
		return false
	} else {
		t.Error("for nil input, expected ErrPublicationInvalid but got nil error")
		return false
	}
}
