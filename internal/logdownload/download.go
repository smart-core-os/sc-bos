// Package logdownload provides shared utilities for serving rotating log files
// over HTTP with short-lived signed download tokens.
// It is used by both the log system plugin and the audit log subsystem.
package logdownload

import (
	"archive/zip"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
)

// NewHMACKey generates a random 32-byte HMAC key for signing download tokens.
func NewHMACKey() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("logdownload: failed to generate HMAC key: " + err.Error())
	}
	return key
}

// LogAllowedDir returns the directory under which log files are expected,
// derived from config. The result is used by ServeLogDownload to validate
// paths embedded in download tokens before opening them.
func LogAllowedDir(logFilePath, logDir string) string {
	if logDir != "" {
		return filepath.Clean(logDir)
	}
	if logFilePath != "" {
		return filepath.Clean(filepath.Dir(logFilePath))
	}
	return ""
}

// GetDownloadLogURL generates a signed download URL for the log file(s) matching
// glob / logDir. urlBase is prepended to downloadPath when forming absolute URLs.
func GetDownloadLogURL(
	req *logpb.GetDownloadLogUrlRequest,
	glob, logDir, urlBase, downloadPath string,
	key []byte,
	ttl time.Duration,
) (*logpb.GetDownloadLogUrlResponse, error) {
	if req.IncludeRotated {
		var files []string
		if logDir != "" {
			entries, err := os.ReadDir(logDir)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "read log dir: %v", err)
			}
			for _, e := range entries {
				if !e.Type().IsRegular() {
					continue
				}
				files = append(files, filepath.Join(logDir, e.Name()))
			}
		} else {
			var err error
			files, err = filepath.Glob(glob)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "glob log files: %v", err)
			}
		}
		if len(files) == 0 {
			return &logpb.GetDownloadLogUrlResponse{}, nil
		}
		tokenStr, err := signDownloadToken(downloadClaims{ZipFiles: files}, key, time.Now().Add(ttl))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "sign token: %v", err)
		}
		u, err := buildDownloadURL(urlBase, downloadPath, tokenStr)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "build url: %v", err)
		}
		return &logpb.GetDownloadLogUrlResponse{Files: []*logpb.GetDownloadLogUrlResponse_LogFile{{
			Url:         u,
			Filename:    "logs.zip",
			ContentType: "application/zip",
		}}}, nil
	}

	files, err := filepath.Glob(glob)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "glob log files: %v", err)
	}
	if len(files) == 0 {
		return &logpb.GetDownloadLogUrlResponse{}, nil
	}
	fp := files[len(files)-1]
	info, err := os.Stat(fp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "stat log file: %v", err)
	}
	tokenStr, err := signDownloadToken(downloadClaims{FilePath: fp}, key, time.Now().Add(ttl))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "sign token: %v", err)
	}
	u, err := buildDownloadURL(urlBase, downloadPath, tokenStr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build url: %v", err)
	}
	return &logpb.GetDownloadLogUrlResponse{Files: []*logpb.GetDownloadLogUrlResponse_LogFile{{
		Url:         u,
		Filename:    filepath.Base(fp),
		SizeBytes:   info.Size(),
		ContentType: DetectContentType(fp),
	}}}, nil
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

// ServeLogDownload is the HTTP handler for log file downloads.
// It validates the signed JWT in the "dlt" query parameter and streams
// the requested file or zip archive. key is the HMAC signing key;
// allowedDir restricts which files may be served (empty = unrestricted).
func ServeLogDownload(w http.ResponseWriter, r *http.Request, key []byte, allowedDir string) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")

	tokenB64 := r.URL.Query().Get("dlt")
	if tokenB64 == "" {
		http.Error(w, "missing download token", http.StatusUnauthorized)
		return
	}
	tokenStr, err := base64.RawURLEncoding.DecodeString(tokenB64)
	if err != nil {
		http.Error(w, "invalid download token", http.StatusUnauthorized)
		return
	}
	claims, err := parseDownloadToken(string(tokenStr), key)
	if err != nil {
		http.Error(w, "invalid or expired download token", http.StatusUnauthorized)
		return
	}

	if len(claims.ZipFiles) > 0 {
		serveZipDownload(w, claims.ZipFiles, allowedDir)
		return
	}

	fp := filepath.Clean(claims.FilePath)
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

// --------------------------------------------------------------------------
// Internal helpers
// --------------------------------------------------------------------------

type downloadClaims struct {
	FilePath string   `json:"fp,omitempty"`
	ZipFiles []string `json:"zf,omitempty"`
}

type downloadJWTClaims struct {
	josejwt.Claims
	Body downloadClaims `json:"b"`
}

func signDownloadToken(body downloadClaims, key []byte, expiry time.Time) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.HS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		return "", err
	}
	claims := downloadJWTClaims{
		Claims: josejwt.Claims{Expiry: josejwt.NewNumericDate(expiry)},
		Body:   body,
	}
	return josejwt.Signed(signer).Claims(claims).Serialize()
}

func parseDownloadToken(tokenStr string, key []byte) (*downloadClaims, error) {
	tok, err := josejwt.ParseSigned(tokenStr, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return nil, err
	}
	var claims downloadJWTClaims
	if err := tok.Claims(key, &claims); err != nil {
		return nil, err
	}
	if err := claims.ValidateWithLeeway(josejwt.Expected{Time: time.Now()}, time.Minute); err != nil {
		return nil, err
	}
	return &claims.Body, nil
}

func buildDownloadURL(urlBase, downloadPath, tokenStr string) (string, error) {
	base := urlBase
	if base == "" {
		base = downloadPath
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(u.Path, "/") {
		u.Path = downloadPath
	} else if u.Path == "" {
		u.Path = downloadPath
	}
	q := u.Query()
	q.Set("dlt", base64.RawURLEncoding.EncodeToString([]byte(tokenStr)))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func serveZipDownload(w http.ResponseWriter, files []string, allowedDir string) {
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

// isUnderDir reports whether fp resides directly under dir.
// An empty dir disables the check (returns true).
func isUnderDir(fp, dir string) bool {
	if dir == "" {
		return true
	}
	return strings.HasPrefix(fp, dir+string(filepath.Separator))
}
