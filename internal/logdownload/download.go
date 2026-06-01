// Package logdownload provides shared utilities for serving log-style file
// downloads over the internal/download signed-URL router.
//
// Consumers (pkg/system/log, internal/audit) register a FileDownloadHandler
// on the shared router and mint URLs through GenerateFileDownloadURL. The
// remaining exported helpers cover file resolution, content-type detection,
// allowed-dir validation, and ring-buffer metadata refresh — all
// log-specific concerns that fall outside the generic download router.
package logdownload

import (
	"archive/zip"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
)

// LogAllowedDir returns the directory under which log files are expected,
// derived from config. The result is used to validate paths embedded in
// signed download tokens before opening them.
func LogAllowedDir(logFilePath, logDir string) string {
	if logDir != "" {
		return filepath.Clean(logDir)
	}
	if logFilePath != "" {
		return filepath.Clean(filepath.Dir(logFilePath))
	}
	return ""
}

// ResolveLatestLogFile returns the lexically-last file matching glob.
// Returns "" with no error when nothing matches. Lumberjack-style rotated files
// have lexically-ordered timestamp suffixes so the last match is the most
// recent file.
func ResolveLatestLogFile(glob string) (string, error) {
	files, err := filepath.Glob(glob)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", nil
	}
	return files[len(files)-1], nil
}

// ResolveRotatedLogFiles returns the set of log files to bundle into a zip.
// When logDir is set, lists all regular files in it; otherwise resolves glob.
// Returns nil when no files are found.
func ResolveRotatedLogFiles(glob, logDir string) ([]string, error) {
	if logDir != "" {
		entries, err := os.ReadDir(logDir)
		if err != nil {
			return nil, err
		}
		var files []string
		for _, e := range entries {
			if !e.Type().IsRegular() {
				continue
			}
			files = append(files, filepath.Join(logDir, e.Name()))
		}
		return files, nil
	}
	return filepath.Glob(glob)
}

// RefreshMetadata scans the log files matching glob / logDir and updates the
// model's LogMetadata with their total size and count.
func RefreshMetadata(model *logpb.Model, glob, logDir string, logger *zap.Logger) {
	var totalSize int64
	var fileCount int

	if logDir != "" {
		entries, err := os.ReadDir(logDir)
		if err != nil {
			logger.Warn("failed to read log dir", zap.String("dir", logDir), zap.Error(err))
			return
		}
		for _, e := range entries {
			if !e.Type().IsRegular() {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			totalSize += info.Size()
			fileCount++
		}
	} else {
		files, err := filepath.Glob(glob)
		if err != nil {
			logger.Warn("failed to glob log files", zap.String("glob", glob), zap.Error(err))
			return
		}
		for _, f := range files {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}
			totalSize += info.Size()
			fileCount++
		}
	}

	_, _ = model.UpdateLogMetadata(&logpb.LogMetadata{
		TotalSizeBytes: totalSize,
		FileCount:      int32(fileCount),
	})
}

// ServeFile streams filePath as an HTTP download response, setting
// Content-Type, Content-Disposition, and Content-Length. allowedDir restricts
// which files may be served (empty disables the check).
func ServeFile(w http.ResponseWriter, _ *http.Request, filePath, allowedDir string) {
	fp := filepath.Clean(filePath)
	if !isUnderDir(fp, allowedDir) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	f, err := os.Open(fp)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.Error(w, "not a file", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", DetectContentType(fp))
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(fp)+`"`)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	_, _ = io.Copy(w, f)
}

// ServeZip streams a zip archive containing files as the HTTP response. Each
// file is filtered through allowedDir before inclusion.
func ServeZip(w http.ResponseWriter, files []string, allowedDir string) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="logs.zip"`)

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, fp := range files {
		fp = filepath.Clean(fp)
		if !isUnderDir(fp, allowedDir) {
			continue
		}
		f, err := os.Open(fp)
		if err != nil {
			continue
		}
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			f.Close()
			continue
		}
		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			f.Close()
			continue
		}
		fh.Name = filepath.Base(fp)
		fh.Method = zip.Deflate
		fw, err := zw.CreateHeader(fh)
		if err != nil {
			f.Close()
			continue
		}
		_, _ = io.Copy(fw, f)
		f.Close()
	}
}

// DetectContentType returns the MIME type for the given filename based on its extension.
func DetectContentType(fp string) string {
	switch {
	case strings.HasSuffix(fp, ".gz"):
		return "application/gzip"
	case strings.HasSuffix(fp, ".zst"):
		return "application/zstd"
	case strings.HasSuffix(fp, ".bz2"):
		return "application/x-bzip2"
	default:
		ct := mime.TypeByExtension(filepath.Ext(fp))
		if ct == "" {
			return "text/plain"
		}
		return ct
	}
}

// isUnderDir reports whether fp resides directly under dir.
// An empty dir disables the check (returns true).
func isUnderDir(fp, dir string) bool {
	if dir == "" {
		return true
	}
	return strings.HasPrefix(fp, dir+string(filepath.Separator))
}
