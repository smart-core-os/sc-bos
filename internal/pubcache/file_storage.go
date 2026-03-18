package pubcache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/publicationpb"
)

// NewFileStorage returns a Storage that persists publications in a directory on the filesystem.
// The directory must already exist, and we must have the necessary permissions to create, list and delete files
// in the directory.
// This implementation doesn't use the contexts passed to methods.
func NewFileStorage(baseDir string) Storage {
	return &fileStorage{baseDir: baseDir}
}

type fileStorage struct {
	baseDir string
}

func (f *fileStorage) LoadPublication(_ context.Context, pubID string) (*publicationpb.Publication, error) {
	filename, ok := idToFilename(pubID)
	if !ok {
		return nil, ErrPublicationNotFound
	}
	contents, err := os.ReadFile(filepath.Join(f.baseDir, filename))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrPublicationNotFound
	} else if err != nil {
		return nil, err
	}

	pub := &publicationpb.Publication{}
	err = proto.Unmarshal(contents, pub)
	if err != nil {
		return nil, err
	}

	// check the publication has the expected ID
	if pub.Id != pubID {
		return nil, fmt.Errorf("stored publication mismatch: expected %q got %q", pubID, pub.Id)
	}
	return pub, nil
}

func (f *fileStorage) StorePublication(_ context.Context, pub *publicationpb.Publication) error {
	if pub.GetId() == "" {
		return ErrPublicationInvalid
	}

	encoded, err := proto.Marshal(pub)
	if err != nil {
		return err
	}

	filename, ok := idToFilename(pub.Id)
	if !ok {
		return ErrPublicationInvalid
	}

	// create a temporary file
	temp, err := os.CreateTemp(f.baseDir, filename+".*")
	if err != nil {
		return err
	}
	defer func() {
		_ = temp.Close()           // ensure closed
		_ = os.Remove(temp.Name()) // remove the file, if it wasn't renamed successfully
	}()

	// write the encoded protobuf and ensure file closes successfully
	_, err = temp.Write(encoded)
	if err != nil {
		return err
	}
	err = temp.Close()
	if err != nil {
		return err
	}

	// rename the temporary file to the intended filename
	return os.Rename(temp.Name(), filepath.Join(f.baseDir, filename))
}

func (f *fileStorage) ListPublications(_ context.Context) (pubIDs []string, err error) {
	entries, err := os.ReadDir(f.baseDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.Type() != 0 {
			// skip anything that's not a regular file
			continue
		}

		id, ok := filenameToId(entry.Name())
		if !ok {
			// skip filenames we don't understand (might be an OS metadata file or something)
			continue
		}

		pubIDs = append(pubIDs, id)
	}
	return
}

func (f *fileStorage) DeletePublication(_ context.Context, pubID string) (present bool, err error) {
	filename, ok := idToFilename(pubID)
	if !ok {
		return false, ErrPublicationInvalid
	}

	err = os.Remove(filepath.Join(f.baseDir, filename))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func idToFilename(id string) (filename string, ok bool) {
	if len(id) == 0 {
		return "", false
	}

	var builder strings.Builder
	var err error

	for _, b := range []byte(id) {
		if filenameByteAllowed(b) {
			err = builder.WriteByte(b)
		} else {
			// perform string escaping
			_, err = fmt.Fprintf(&builder, "$%02X", b)
		}

		if err != nil {
			// shouldn't be possible
			panic("unexpected error")
		}
	}

	return builder.String(), true
}

func filenameToId(filename string) (id string, ok bool) {
	if len(filename) == 0 {
		return "", false
	}

	var builder strings.Builder

	remaining := bytes.NewBufferString(filename)
	for {
		b, err := remaining.ReadByte()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			// no other kind of error should be possible
			panic("unexpected error")
		}

		if filenameByteAllowed(b) {
			err = builder.WriteByte(b)
		} else if b == '$' {
			_, err = fmt.Fscanf(remaining, "%02X", &b)
		} else {
			// an unexpected character
			return "", false
		}

		if err != nil {
			// if an error occurred, the input is invalid
			return "", false
		}
	}

	return builder.String(), true
}

func filenameByteAllowed(b byte) bool {
	return (b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z') ||
		b == '-' ||
		b == '_' ||
		b == ' '
}
