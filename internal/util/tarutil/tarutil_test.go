package tarutil

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
)

type entry struct {
	name     string
	typeflag byte
	mode     int64
	data     string
}

func dir(name string) entry { return entry{name: name, typeflag: tar.TypeDir, mode: 0755} }

func file(name, data string) entry {
	return entry{name: name, typeflag: tar.TypeReg, mode: 0644, data: data}
}

func makeTarGZ(t *testing.T, entries ...entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)
	for _, e := range entries {
		header := &tar.Header{
			Name:     e.name,
			Mode:     e.mode,
			Typeflag: e.typeflag,
			Size:     int64(len(e.data)),
		}
		if e.typeflag == tar.TypeSymlink {
			header.Linkname = e.data
			header.Size = 0
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if header.Size > 0 {
			if _, err := tarWriter.Write([]byte(e.data)); err != nil {
				t.Fatalf("write tar content: %v", err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}

func genLargeFileTarGZ(t *testing.T, size int64) []byte {
	t.Helper()
	contents := bytes.Repeat([]byte{0xDE, 0xAD, 0xBE, 0xEF}, int((size+3)/4))[:size]
	return makeTarGZ(t, file("file.bin", string(contents)))
}

func openRoot(t *testing.T) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatalf("open root: %v", err)
	}
	t.Cleanup(func() { _ = root.Close() })
	return root
}

func assertFileContent(t *testing.T, fsys fs.FS, path, want string) {
	t.Helper()
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		t.Fatalf("read file %q: %v", path, err)
	}
	if got := string(data); got != want {
		t.Errorf("file %q: got %q, want %q", path, got, want)
	}
}

func TestExtractTarGZ(t *testing.T) {
	t.Run("files and directories", func(t *testing.T) {
		data := makeTarGZ(t,
			file("config.json", `{"key":"value"}`),
			dir("sub"),
			file("sub/nested.txt", "nested content"),
		)
		root := openRoot(t)

		if err := ExtractTarGZ(bytes.NewReader(data), root, -1); err != nil {
			t.Fatalf("ExtractTarGZ: %v", err)
		}

		fsys := root.FS()
		assertFileContent(t, fsys, "config.json", `{"key":"value"}`)
		assertFileContent(t, fsys, "sub/nested.txt", "nested content")
	})

	t.Run("empty archive", func(t *testing.T) {
		data := makeTarGZ(t)
		root := openRoot(t)

		if err := ExtractTarGZ(bytes.NewReader(data), root, -1); err != nil {
			t.Errorf("expected no error for empty archive, got: %v", err)
		}
	})

	t.Run("invalid gzip", func(t *testing.T) {
		root := openRoot(t)

		err := ExtractTarGZ(bytes.NewReader([]byte("not gzip")), root, -1)
		if err == nil {
			t.Error("expected error for invalid gzip, got nil")
		}
	})

	t.Run("unsupported entry type (symlink)", func(t *testing.T) {
		data := makeTarGZ(t, entry{name: "link", typeflag: tar.TypeSymlink, mode: 0644, data: "target"})
		root := openRoot(t)

		err := ExtractTarGZ(bytes.NewReader(data), root, -1)
		if err == nil {
			t.Fatal("expected error for symlink entry, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported") || !strings.Contains(err.Error(), "link") {
			t.Errorf("expected error to name the unsupported entry, got: %v", err)
		}
	})

	t.Run("truncated tar", func(t *testing.T) {
		data := makeTarGZ(t,
			file("config.json", `{"key":"value"}`),
			dir("sub"),
			file("sub/nested.txt", "nested content"),
		)
		data = data[:len(data)/2]
		root := openRoot(t)

		err := ExtractTarGZ(bytes.NewReader(data), root, -1)
		if err == nil {
			t.Error("expected error for truncated tar, got nil")
		}
	})

	t.Run("file without parent directory entry", func(t *testing.T) {
		data := makeTarGZ(t, file("sub/nested.txt", "nested content"))
		root := openRoot(t)

		err := ExtractTarGZ(bytes.NewReader(data), root, -1)
		if err == nil {
			t.Error("expected error for file without parent directory entry, got nil")
		}
	})

	t.Run("escape root", func(t *testing.T) {
		data := makeTarGZ(t, file("../escaped.txt", "escaped"))
		root := openRoot(t)

		err := ExtractTarGZ(bytes.NewReader(data), root, -1)
		if err == nil {
			t.Error("expected error for entry outside root, got nil")
		}
	})

	t.Run("over size limit", func(t *testing.T) {
		// with the tar headers included, the archive exceeds the size limit
		data := genLargeFileTarGZ(t, 1024)
		root := openRoot(t)

		err := ExtractTarGZ(bytes.NewReader(data), root, 1024)
		if !errors.Is(err, ErrSizeLimit) {
			t.Errorf("expected error wrapping ErrSizeLimit, got: %v", err)
		}
	})

	t.Run("within size limit", func(t *testing.T) {
		data := genLargeFileTarGZ(t, 1024)
		root := openRoot(t)

		if err := ExtractTarGZ(bytes.NewReader(data), root, 1024*1024); err != nil {
			t.Errorf("expected no error within size limit, got: %v", err)
		}
	})
}
