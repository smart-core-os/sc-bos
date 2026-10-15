// Package tarutil extracts tar archives into a directory.
package tarutil

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrSizeLimit is returned, wrapped, when the decompressed archive exceeds the limit.
var ErrSizeLimit = errors.New("archive exceeds size limit")

// ExtractTarGZ extracts the gzipped tar stream src into dst.
// Only directories and regular files are supported; any other entry type results in an error.
// Directories are created with mode 0755. File modes are as in the archive, but masked with 0755, and 0600 (owner
// read/write) always allowed.
// Each file's parent directory must already exist or appear earlier in the archive.
// If maxSize is positive, it limits the decompressed size of the archive. If the decompressed archive size exceeds
// this, the extraction will be aborted (with the incomplete extracted tree remaining in dst).
func ExtractTarGZ(src io.Reader, dst *os.Root, maxSize int64) error {
	srcUncompressed, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer func() {
		_ = srcUncompressed.Close()
	}()

	var srcLimited io.Reader = srcUncompressed
	if maxSize > 0 {
		srcLimited = &limitedReader{R: srcUncompressed, N: maxSize}
	}

	tarReader := tar.NewReader(srcLimited)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, ErrSizeLimit) {
			return sizeLimitError(maxSize)
		} else if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err = dst.MkdirAll(header.Name, 0755); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
		case tar.TypeReg:
			err = copyFile(dst, header.Name, os.FileMode(header.Mode)&0755|0600, tarReader)
			if errors.Is(err, ErrSizeLimit) {
				return sizeLimitError(maxSize)
			} else if err != nil {
				return fmt.Errorf("copy file: %w", err)
			}
		default:
			return fmt.Errorf("unsupported tar entry type: %c in file %s", header.Typeflag, header.Name)
		}
	}
	return nil
}

func copyFile(root *os.Root, path string, mode os.FileMode, src io.Reader) error {
	dstFile, err := root.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, src); err != nil {
		return fmt.Errorf("copy file contents: %w", err)
	}
	return nil
}

// limitedReader is like io.LimitedReader, but returns ErrSizeLimit instead of EOF once N bytes have been read.
type limitedReader struct {
	R io.Reader
	N int64
}

func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, ErrSizeLimit
	}
	if int64(len(p)) > l.N {
		p = p[:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return n, err
}

func sizeLimitError(maxSize int64) error {
	return fmt.Errorf("%w of %d bytes", ErrSizeLimit, maxSize)
}
